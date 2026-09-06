import request from '@/utils/http'
/** 分页后的缓存 value 响应（后端 response.CacheValuePage，snake_case JSON 字段） */
export interface CacheValuePage {
  key: string
  type: string
  ttl: number
  total: number
  start: number
  has_more: boolean
  /** set/hash 续扫游标：后端 json:",string" 编码，防 JS 大整数精度丢失 */
  next_cursor: string
  truncated: boolean
  value: unknown
}

/** zset 成员条目 */
export interface ZSetEntry {
  member: string
  score: number
}

/** hash 字段条目 */
export interface HashEntry {
  field: string
  value: string
}

/** 分页取值参数：list/zset 用 offset+limit；set/hash 用 cursor+limit；string 用 limit 字节窗 */
export interface FetchPageOptions {
  key: string
  offset?: number
  limit?: number
  /** 首页请求传 0，续扫回传响应字符串 */
  cursor?: string | number
}

export type FetchPage = (params: FetchPageOptions) => Promise<CacheValuePage>

const PREFIX = import.meta.env.VITE_API_PREFIX

/** 缓存 key 列表查询参数 */
export interface CacheKeysParams {
  prefix?: string
  cursor?: number
  count?: number
}

/** 单个缓存 key（含 Redis 值类型） */
export interface CacheKeyInfo {
  key: string
  type: string
}

/** 缓存 key 列表响应 */
export interface CacheKeysResult {
  keys: CacheKeyInfo[]
  cursor: string
}

/** 批量删除缓存参数 */
export interface DeleteCacheKeysParams {
  prefix?: string
  maxCount?: number
}

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

/** 时序快照中的单个时间桶 */
export interface SqlHistoryPoint {
  timestamp: string
  count: number
  qps: number
  avg_ms: number
  p50_ms: number
  p95_ms: number
  p99_ms: number
  max_ms: number
  error_count: number
  slow_count: number
}

/** SQL 监控时序快照 */
export interface SqlHistory {
  window_seconds: number
  step_seconds: number
  slow_threshold_ms: number
  recent_qps: number
  buckets: SqlHistoryPoint[]
}

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

/** SQL 监控累计统计 */
export function fetchSQLStats(params: SqlStatsParams = {}) {
  return request.get<SqlStats>({ url: `${PREFIX}/monitor/sql/stats`, params })
}

/** SQL 监控时序数据（QPS/耗时趋势） */
export function fetchSQLHistory(params: SqlHistoryParams) {
  return request.get<SqlHistory>({ url: `${PREFIX}/monitor/sql/history`, params })
}

/** 缓存 key 列表 */
export function fetchCacheKeys(params: CacheKeysParams) {
  return request.get<CacheKeysResult>({ url: `${PREFIX}/monitor/cache/keys`, params })
}

/** 分页查询单个缓存 key 的 value（list/zset: offset+limit 分页；set/hash: cursor+limit 游标；string: limit 字节窗） */
export function fetchCacheValuePage(params: FetchPageOptions) {
  return request.get<CacheValuePage>({
    url: `${PREFIX}/monitor/cache/keys/value`,
    params
  })
}

/** 按前缀批量删除缓存 key */
export function deleteCacheKeys(params: DeleteCacheKeysParams) {
  return request.del<{ deleted: number }>({ url: `${PREFIX}/monitor/cache/keys`, params })
}

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
