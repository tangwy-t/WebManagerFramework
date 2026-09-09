/** system-monitor · pprof 域 API/类型。契约随后端 OpenAPI(snake_case),与后端 response/dto 对齐 */
import request from '@/utils/http'

const PREFIX = import.meta.env.VITE_API_PREFIX

// ════════════════════════ pprof 性能分析 ════════════════════════

/** pprof profile 概览条目 */
export interface PprofProfileEntry {
  name: string
  description: string
  /** snapshot:可即时采样解析;capture:需按需采集下载 */
  category: 'snapshot' | 'capture'
  unit: string
  count: number
}

/** pprof 运行状态概览 */
export interface PprofStatus {
  enabled: boolean
  /** 0 表示不自动关闭 */
  autoOffSeconds: number
  /** >0 时服务端自动关闭倒计时生效 */
  remainingSeconds: number
  profiles: PprofProfileEntry[]
}

/** 火焰图树节点 */
export interface PprofFlameNode {
  name: string
  value: number
  children?: PprofFlameNode[]
}

/** 热点函数一行 */
export interface PprofTopFunc {
  /** 完整函数名 */
  fn: string
  /** 缩短展示名 */
  name: string
  file: string
  line: number
  flat: number
  cum: number
}

/** 单个 profile 的火焰树 + 热点函数 */
export interface PprofFlameData {
  name: string
  unit: string
  sampleType: string
  sampleCount: number
  totalValue: number
  /** 火焰树是否因深度/节点数限制被裁剪 */
  truncated: boolean
  flame: PprofFlameNode | null
  top: PprofTopFunc[]
}

/** 查询 pprof 运行状态 */
export function fetchPprofStatus() {
  return request.get<PprofStatus>({ url: `${PREFIX}/monitor/pprof/status` })
}

/** 启用 pprof 采集 */
export function enablePprof() {
  return request.post<{ enabled: boolean }>({ url: `${PREFIX}/monitor/pprof/enable` })
}

/** 停用 pprof 采集 */
export function disablePprof() {
  return request.post<{ enabled: boolean }>({ url: `${PREFIX}/monitor/pprof/disable` })
}

/** 解析指定 profile 的火焰树与热点函数 */
export function fetchPprofFlame(name: string, top = 15) {
  return request.get<PprofFlameData>({
    url: `${PREFIX}/monitor/pprof/profile/${name}`,
    params: { top }
  })
}

/** 下载 pprof 原始数据（携带鉴权头）。
 *  name 为 snapshot 名(如 heap)时自动附加 debug=0(二进制 proto)；
 *  为 profile/trace 时传递 seconds 等采集参数即可。 */
export function downloadPprofRaw(name: string, extraParams: Record<string, string | number> = {}) {
  const params: Record<string, string | number> =
    name === 'profile' || name === 'trace' ? { ...extraParams } : { debug: 0, ...extraParams }
  // CPU profile/trace 由服务端阻塞采集 seconds 秒,超时需覆盖默认 15s
  return request.download({ url: `${PREFIX}/monitor/debug/pprof/${name}`, params, timeout: 180000 })
}
