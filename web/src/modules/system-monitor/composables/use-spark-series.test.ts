import { describe, expect, it } from 'vitest'
import { sparkWindowCount, buildSparkSeries, SPARK_WINDOW_SECONDS } from './use-spark-series'

describe('sparkWindowCount', () => {
  it('step 3s 与 12s 时按 10 分钟窗口取点并封顶 60', () => {
    expect(sparkWindowCount(3)).toBe(60) // ceil(600/3)=200 → cap 60
    expect(sparkWindowCount(12)).toBe(50)
    expect(sparkWindowCount(60)).toBe(10)
    expect(sparkWindowCount(240)).toBe(3)
  })

  it('窗口参数与上下限生效', () => {
    expect(sparkWindowCount(300, 600)).toBe(3) // ceil(600/300)=2 → min 3
    expect(sparkWindowCount(3, 3600, 1, 1000)).toBe(1000)
  })

  it('非法 step 回退 min', () => {
    expect(sparkWindowCount(0)).toBe(3)
    expect(sparkWindowCount(-5)).toBe(3)
  })
})

interface Point {
  timestamp: string
  cpu: number | null
  memSys: number | null
}

const ts = (s: number) => `2026-09-09 10:00:${String(s % 60).padStart(2, '0')}`

describe('buildSparkSeries', () => {
  it('取桶尾并向前填充缺值、跳过前导 undefined', () => {
    const buckets: Point[] = [
      { timestamp: ts(1), cpu: 1, memSys: null },
      { timestamp: ts(2), cpu: null, memSys: null },
      { timestamp: ts(3), cpu: 3, memSys: 5 }
    ]
    expect(buildSparkSeries(buckets, 3, 'cpu', SPARK_WINDOW_SECONDS)).toEqual([1, 1, 3])
    expect(buildSparkSeries(buckets, 3, 'memSys', SPARK_WINDOW_SECONDS)).toEqual([5])
  })

  it('桶多于窗口点数时只取尾部', () => {
    const buckets = Array.from({ length: 80 }, (_, i) => ({ timestamp: ts(i), cpu: i, memSys: null }))
    const got = buildSparkSeries(buckets, 10, 'cpu', SPARK_WINDOW_SECONDS)
    expect(got[0]).toBe(80 - 60)
    expect(got.length).toBe(60)
  })

  it('空桶返回空数组', () => {
    expect(buildSparkSeries([], 10, 'cpu', SPARK_WINDOW_SECONDS)).toEqual([])
  })
})