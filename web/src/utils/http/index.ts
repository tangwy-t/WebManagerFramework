/**
 * HTTP 请求封装模块（对接 Go 后端统一契约）
 *
 * - 成功：HTTP 200 + `{ code: 0, msg, data }`
 * - 失败：HTTP 非 2xx + `{ code: apperror 业务码, msg, data }`
 * - 鉴权：`Authorization: Bearer <accessToken>`
 * - 401（未授权 / token 过期）自动 refresh 并重放（单飞）
 */
import axios, { AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { useUserStore } from '@/store/modules/user'
import { ApiStatus } from './status'
import { HttpError, handleError, showError, showSuccess } from './error'
import { BaseResponse } from '@/types'

/** 请求配置常量 */
const REQUEST_TIMEOUT = 15000
const LOGOUT_DELAY = 500
const MAX_RETRIES = 0
const RETRY_DELAY = 1000
const UNAUTHORIZED_DEBOUNCE_TIME = 3000

/** 401 防抖状态 */
let isUnauthorizedErrorShown = false
let unauthorizedTimer: NodeJS.Timeout | null = null

/** 扩展 AxiosRequestConfig */
interface ExtendedAxiosRequestConfig extends AxiosRequestConfig {
  showErrorMessage?: boolean
  showSuccessMessage?: boolean
}

const { VITE_API_URL, VITE_WITH_CREDENTIALS, VITE_API_PREFIX } = import.meta.env

const AUTH_REFRESH_PATH = `${VITE_API_PREFIX}/refresh`

/** token 刷新单飞 */
let isRefreshing = false
let pendingQueue: Array<{ resolve: (token: string) => void; reject: (e: unknown) => void }> = []

/** Axios 实例 */
const axiosInstance = axios.create({
  timeout: REQUEST_TIMEOUT,
  baseURL: VITE_API_URL,
  withCredentials: VITE_WITH_CREDENTIALS === 'true',
  validateStatus: (status) => status >= 200 && status < 300,
  transformResponse: [
    (data, headers) => {
      const contentType = headers['content-type']
      if (typeof contentType === 'string' && contentType.includes('application/json')) {
        try {
          return JSON.parse(data)
        } catch {
          return data
        }
      }
      return data
    }
  ]
})

/** 请求拦截器 */
axiosInstance.interceptors.request.use(
  (request: InternalAxiosRequestConfig) => {
    const { accessToken } = useUserStore()
    if (accessToken) request.headers.set('Authorization', `Bearer ${accessToken}`)

    if (request.data && !(request.data instanceof FormData) && !request.headers['Content-Type']) {
      request.headers.set('Content-Type', 'application/json')
      request.data = JSON.stringify(request.data)
    }

    return request
  },
  (error) => {
    showError(createHttpError('请求配置错误', ApiStatus.error))
    return Promise.reject(error)
  }
)

/** 响应拦截器 */
axiosInstance.interceptors.response.use(
  (response: AxiosResponse<BaseResponse>) => {
    // blob 下载走原始二进制数据,不做 {code,msg,data} 信封解包
    if (response.config.responseType === 'blob') return response
    const { code, msg } = response.data
    if (code === ApiStatus.success) return response
    throw createHttpError(msg || '请求失败', code)
  },
  async (error) => {
    if (error.response?.status === ApiStatus.unauthorized) {
      // 仅在已登录（存在 refreshToken）时刷新重放；登录/验证码等无 token 场景直接报错
      if (useUserStore().refreshToken) {
        const config = error.config as InternalAxiosRequestConfig
        const newToken = await refreshAndReplay(config)
        if (newToken) return axiosInstance.request(config)
        handleUnauthorizedError(error.response?.data?.msg || '未授权访问，请重新登录')
      }
    }
    return Promise.reject(handleError(error))
  }
)

/** 统一创建 HttpError */
function createHttpError(message: string, code: number) {
  return new HttpError(message, code)
}

/** 处理 401 错误（refresh 失败后登出并提示） */
function handleUnauthorizedError(message?: string): never {
  const error = createHttpError(message || '未授权访问，请重新登录', ApiStatus.unauthorized)

  if (!isUnauthorizedErrorShown) {
    isUnauthorizedErrorShown = true
    logOut()

    unauthorizedTimer = setTimeout(resetUnauthorizedError, UNAUTHORIZED_DEBOUNCE_TIME)

    showError(error, true)
  }

  throw error
}

/** 重置 401 防抖状态 */
function resetUnauthorizedError() {
  isUnauthorizedErrorShown = false
  if (unauthorizedTimer) clearTimeout(unauthorizedTimer)
  unauthorizedTimer = null
}

/** 退出登录函数 */
function logOut() {
  setTimeout(() => {
    useUserStore().logOut()
  }, LOGOUT_DELAY)
}

/** 刷新访问令牌 */
async function refreshAccessToken(): Promise<string> {
  const { refreshToken } = useUserStore()
  if (!refreshToken) throw new Error('no refresh token')
  const fresh = await axios.post<BaseResponse<Api.Auth.TokenResp>>(AUTH_REFRESH_PATH, {
    refreshToken
  })
  if (fresh.data.code === ApiStatus.success) {
    const { accessToken, refreshToken: nextRefresh } = fresh.data.data
    useUserStore().setToken(accessToken, nextRefresh)
    return accessToken
  }
  throw new Error('refresh failed')
}

function flushPending(token: string) {
  pendingQueue.forEach((p) => p.resolve(token))
  pendingQueue = []
}

function failPending(err: unknown) {
  pendingQueue.forEach((p) => p.reject(err))
  pendingQueue = []
}

/** 401 后刷新并重放原请求；失败返回 null */
async function refreshAndReplay(config?: InternalAxiosRequestConfig): Promise<string | null> {
  if (!config || config.url?.includes('/refresh')) return null
  if (isRefreshing) {
    return new Promise<string>((resolve, reject) => pendingQueue.push({ resolve, reject }))
  }
  isRefreshing = true
  try {
    const token = await refreshAccessToken()
    flushPending(token)
    return token
  } catch (e) {
    failPending(e)
    return null
  } finally {
    isRefreshing = false
  }
}

/** 是否需要重试 */
function shouldRetry(statusCode: number) {
  return [
    ApiStatus.requestTimeout,
    ApiStatus.internalServerError,
    ApiStatus.badGateway,
    ApiStatus.serviceUnavailable,
    ApiStatus.gatewayTimeout
  ].includes(statusCode)
}

/** 请求重试逻辑 */
async function retryRequest<T>(
  config: ExtendedAxiosRequestConfig,
  retries: number = MAX_RETRIES
): Promise<T> {
  try {
    return await request<T>(config)
  } catch (error) {
    if (retries > 0 && error instanceof HttpError && shouldRetry(error.code)) {
      await delay(RETRY_DELAY)
      return retryRequest<T>(config, retries - 1)
    }
    throw error
  }
}

/** 延迟函数 */
function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** 请求函数 */
async function request<T = any>(config: ExtendedAxiosRequestConfig): Promise<T> {
  // POST | PUT 参数自动填充
  if (
    ['POST', 'PUT'].includes(config.method?.toUpperCase() || '') &&
    config.params &&
    !config.data
  ) {
    config.data = config.params
    config.params = undefined
  }

  try {
    const res = await axiosInstance.request<BaseResponse<T>>(config)

    if (config.showSuccessMessage && res.data.msg) {
      showSuccess(res.data.msg)
    }

    return res.data.data as T
  } catch (error) {
    if (error instanceof HttpError && error.code !== ApiStatus.unauthorized) {
      const showMsg = config.showErrorMessage !== false
      showError(error, showMsg)
    }
    return Promise.reject(error)
  }
}

/** 下载二进制文件（携带鉴权头,返回 Blob）。
 *  适用于 pprof 原始 profile/trace 等非 JSON 信封的接口。 */
async function download(config: ExtendedAxiosRequestConfig): Promise<Blob> {
  const res = await axiosInstance.request<Blob>({
    ...config,
    responseType: 'blob'
  })
  return res.data
}

/** API 方法集合 */
const api = {
  download,
  get<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'GET' })
  },
  post<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'POST' })
  },
  put<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'PUT' })
  },
  del<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'DELETE' })
  },
  request<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>(config)
  }
}

export default api
