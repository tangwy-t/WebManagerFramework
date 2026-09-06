package sqlhistory

import (
	"fmt"
	"time"

	"github.com/tangwy-t/webmanager-server/internal/pkg/util"
)

// maxAggBuckets 查询聚合最多返回的桶数;超限时自动放大 step,
// 与 ring buffer 版 database.SnapshotHistory 同策略。
const maxAggBuckets = 4000

// bucketAcc 聚合中间累加器。
type bucketAcc struct {
	count                          int64
	avgSum, p50Sum, p95Sum, p99Sum float64
	max                            float64
	errs, slows                    int64
}

// aggregate 把采样点按绝对时间对齐归并为 step 桶(纯函数,便于表驱动单测)。
// avg/p50/p95/p99 采用 count 加权平均 —— 跨 3s 采样合并成大桶时的近似值,
// 可能低估孤立尖峰;监控场景可接受(spec 已注明)。
// 所有派生浮点(QPS/耗时)输出前经 util.Round2 量化到 2 位小数。
// 无样本的桶剔除,不产生误导性零值。
func aggregate(points []Point, window, step time.Duration, now time.Time) (*Snapshot, error) {
	if window <= 0 || step <= 0 {
		return nil, fmt.Errorf("sqlhistory: window 与 step 必须为正数")
	}
	if step < time.Second {
		return nil, fmt.Errorf("sqlhistory: step 最小 1s")
	}
	if step > window {
		return nil, fmt.Errorf("sqlhistory: step 不得超过 window")
	}

	stepSec := int64(step.Seconds())
	windowSec := int64(window.Seconds())
	// 桶数超限时向上放大 step,保证响应体积可控
	if windowSec/stepSec > maxAggBuckets {
		stepSec = (windowSec + maxAggBuckets - 1) / maxAggBuckets
		if stepSec < 1 {
			stepSec = 1
		}
	}
	nBuckets := int(windowSec / stepSec)
	if nBuckets < 1 {
		nBuckets = 1
	}

	cutoff := now.Add(-time.Duration(windowSec) * time.Second)
	t0 := cutoff.Unix()
	t0 -= t0 % stepSec

	raw := make([]bucketAcc, nBuckets)
	recentCut := now.Add(-time.Duration(stepSec) * time.Second)
	var recentCount int64

	for _, p := range points {
		ts := time.UnixMilli(p.T)
		if ts.Before(cutoff) {
			continue
		}
		if !ts.Before(recentCut) {
			recentCount += p.Count
		}
		pos := int((ts.Unix() - t0) / stepSec)
		// 边界桶:与 ring buffer 版 SnapshotHistory 同语义,归入最后一桶
		if pos == nBuckets {
			pos = nBuckets - 1
		}
		if pos < 0 || pos >= nBuckets {
			continue
		}
		b := &raw[pos]
		b.count += p.Count
		b.avgSum += p.AvgMs * float64(p.Count)
		b.p50Sum += p.P50Ms * float64(p.Count)
		b.p95Sum += p.P95Ms * float64(p.Count)
		b.p99Sum += p.P99Ms * float64(p.Count)
		if p.MaxMs > b.max {
			b.max = p.MaxMs
		}
		b.errs += p.ErrorCount
		b.slows += p.SlowCount
	}

	snap := &Snapshot{
		WindowSeconds: windowSec,
		StepSeconds:   stepSec,
		RecentQPS:     util.Round2(float64(recentCount) / float64(stepSec)),
		Buckets:       make([]Bucket, 0, nBuckets),
	}
	for i := range raw {
		b := &raw[i]
		if b.count == 0 {
			continue
		}
		// 所有派生浮点(QPS/加权平均耗时)输出前统一量化到 2 位小数:
		// 采样点即使已量化,加权平均(count 加权)仍会带出长尾小数。
		snap.Buckets = append(snap.Buckets, Bucket{
			Timestamp:  util.JSONTime(time.Unix(t0+int64(i)*stepSec, 0)),
			Count:      b.count,
			QPS:        util.Round2(float64(b.count) / float64(stepSec)),
			AvgMs:      util.Round2(b.avgSum / float64(b.count)),
			P50Ms:      util.Round2(b.p50Sum / float64(b.count)),
			P95Ms:      util.Round2(b.p95Sum / float64(b.count)),
			P99Ms:      util.Round2(b.p99Sum / float64(b.count)),
			MaxMs:      util.Round2(b.max),
			ErrorCount: b.errs,
			SlowCount:  b.slows,
		})
	}
	return snap, nil
}
