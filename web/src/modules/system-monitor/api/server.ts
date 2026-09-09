/** system-monitor · server 域 API/类型。契约随后端 OpenAPI(snake_case),与后端 response/dto 对齐 */
import request from '@/utils/http'

const PREFIX = import.meta.env.VITE_API_PREFIX

/** 服务器监控统计 */
export function fetchServerStats() {
  return request.get<Api.Monitor.ServerStats>({ url: `${PREFIX}/monitor/server/stats` })
}

/** 服务器监控历史查询参数 */
export interface ServerHistoryParams {
  /** 时间窗口,如 15m、1h、6h、24h(后端合法范围 1m~24h) */
  window: string
  /** 分桶粒度,如 3s、12s、60s、240s */
  step: string
}

/** 服务器监控历史采样桶(绝对时间对齐) */
export interface ServerHistoryPoint {
  /** 桶起始时间 */
  timestamp: string
  cpu: number | null
  memSys: number | null
  heapAlloc: number | null
  sysMem: number | null
  goroutines: number | null
  gcNum: number | null
  gcPauseMs: number | null
  disk: number | null
  load1: number | null
  uptime: number | null
}

/** 服务器监控历史查询结果 */
export interface ServerHistory {
  window_seconds: number
  step_seconds: number
  buckets: ServerHistoryPoint[]
}

/** 服务器监控历史时序 */
export function fetchServerHistory(params: ServerHistoryParams) {
  return request.get<ServerHistory>({ url: `${PREFIX}/monitor/server/history`, params })
}
