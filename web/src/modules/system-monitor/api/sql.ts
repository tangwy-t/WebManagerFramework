/** system-monitor · sql 域 API/类型。契约随后端 OpenAPI(snake_case),与后端 response/dto 对齐 */
import request from '@/utils/http'

const PREFIX = import.meta.env.VITE_API_PREFIX

/** 按维度(表/操作类型)的统计 */
export interface SqlDimStats {
  count: number
  avg_ms: number
  max_ms: number
  min_ms: number
  p50_ms?: number
  p95_ms?: number
  p99_ms?: number
}

/** SQL 监控全局累计统计（与后端 snake_case JSON 对齐） */
export interface SqlStats {
  global: {
    count: number
    avg_ms: number
    max_ms: number
    min_ms: number
    p50_ms: number
    p95_ms: number
    p99_ms: number
    error_count: number
    slow_count: number
    slow_threshold_ms: number
  }
  by_table: Record<string, SqlDimStats>
  by_operation: Record<string, SqlDimStats>
  slow_queries: SqlQueryEntry[]
}

/** 单条 SQL 执行记录 */
export interface SqlQueryEntry {
  timestamp: string
  duration_ms: number
  sql: string
  table: string
  operation: string
  is_error?: boolean
  is_slow?: boolean
}

/** 时序快照中的单个时间桶 → Api.Monitor.SQLHistoryPoint(api.generated.d.ts 为唯一事实源) */
export type SqlHistoryPoint = Api.Monitor.SQLHistoryPoint

/** SQL 监控时序快照 → Api.Monitor.SQLHistorySnapshot */
export type SqlHistory = Api.Monitor.SQLHistorySnapshot

/** SQL 监控统计查询参数 */
export interface SqlStatsParams {
  /** 时间窗口（如 5m,1h），留空返回累计统计 */
  window?: string
}

/** SQL 监控时序查询参数 */
export interface SqlHistoryParams {
  window: string
  step: string
}

/** SQL 监控累计统计 */
export function fetchSQLStats(params: SqlStatsParams = {}) {
  return request.get<SqlStats>({ url: `${PREFIX}/monitor/sql/stats`, params })
}

/** SQL 监控时序数据（QPS/耗时趋势） */
export function fetchSQLHistory(params: SqlHistoryParams) {
  return request.get<SqlHistory>({ url: `${PREFIX}/monitor/sql/history`, params })
}
