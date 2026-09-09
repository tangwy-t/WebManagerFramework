/** 迷你趋势固定窗口(秒):与旧 sparkData 的 10m 窗口语义一致 */
export const SPARK_WINDOW_SECONDS = 600

/**
 * 迷你趋势取点数:ceil(窗口/步长),封顶 [min,max]。
 * step 非法(≤0)时回退 min,视图层由 history step_seconds 提供,恒 ≥1。
 */
export function sparkWindowCount(
  stepSeconds: number,
  windowSeconds: number = SPARK_WINDOW_SECONDS,
  min = 3,
  max = 60
): number {
  if (!Number.isFinite(stepSeconds) || stepSeconds <= 0) return min
  const n = Math.ceil(windowSeconds / stepSeconds)
  return Math.min(Math.max(n, min), max)
}

/**
 * 从历史桶尾部派生迷你趋势序列:缺值向前填充,跳过前导 undefined
 * (与旧 server.vue seriesOf 语义一致)。
 */
export function buildSparkSeries<T extends { timestamp: string }>(
  buckets: T[],
  stepSeconds: number,
  key: Extract<keyof T, string>,
  windowSeconds: number = SPARK_WINDOW_SECONDS
): number[] {
  const count = sparkWindowCount(stepSeconds, windowSeconds)
  const tail = buckets.slice(-count)
  const out: number[] = []
  let prev: number | undefined
  for (const b of tail) {
    const v = b[key]
    if (typeof v === 'number' && Number.isFinite(v)) prev = v
    if (prev !== undefined) out.push(prev)
  }
  return out
}