package serverstats

import (
	"fmt"
	"time"

	"github.com/tangwy-t/webmanager-server/internal/pkg/util"
)

// round2 把浮点数量化到 2 位小数(委托 util.Round2,全仓统一精度约定)。
// 采样点已在采集端量化(handler roundTo2),但求平均会重新带出
// 长尾小数(如 1.2333333),聚合输出前必须再量化一次,保证
// /history 响应字段始终是干净的 2 位小数。
func round2(v float64) float64 {
	return util.Round2(v)
}

// bucketAcc 聚合单个时间桶的中间状态。
type bucketAcc struct {
	cpuSum, cpuN     float64
	memSum, memN     float64
	heapSum, heapN   float64
	sysSum, sysN     float64
	gorSum, gorN     float64
	pauseSum, pauseN float64
	diskSum, diskN   float64
	loadSum, loadN   float64

	lastT      int64 // 桶内时间戳最大的采样点
	gcNumLast  uint32
	uptimeLast int64
}

// aggregate 把原始采样点聚合为按绝对时间对齐的时间桶。
// 规则(与设计文档一致):
//   - 桶边界 = epoch 秒对 step 整除,两次轮询桶划分稳定
//   - cpu/memSys/heapAlloc/sysMem/goroutines/gcPauseMs/disk/load1 取均值,
//     缺值点不参与平均;桶内该字段全缺则输出 nil(JSON 省略)
//   - 所有均值输出前统一 round2 量化到 2 位小数(与采集端一致),
//     避免长尾小数(12/3=4 干净,但 1.23+1.22 平均是 1.2250000000000001)
//   - gcNum/uptime 取桶内最新采样点的值(单调递增量,均值无意义)
//   - 空桶剔除;窗口外的点丢弃;恰落在 [now, now+step) 的点归入最后一桶
func aggregate(points []Point, window, step time.Duration, now time.Time) (*Snapshot, error) {
	stepSec := int64(step.Seconds())
	if stepSec < 1 {
		return nil, fmt.Errorf("step must be >= 1s, got %s", step)
	}
	windowSec := int64(window.Seconds())
	if windowSec < stepSec {
		return nil, fmt.Errorf("window must be >= step, got window=%s step=%s", window, step)
	}

	nBuckets := windowSec / stepSec
	if nBuckets > maxHistoryBuckets {
		stepSec = (windowSec + maxHistoryBuckets - 1) / maxHistoryBuckets
		nBuckets = windowSec / stepSec
	}

	cutoff := now.Add(-time.Duration(windowSec) * time.Second)
	t0 := cutoff.Unix()
	t0 -= t0 % stepSec

	accs := make([]bucketAcc, nBuckets)
	for _, p := range points {
		sec := p.T / 1000
		if time.Unix(sec, 0).Before(cutoff) {
			continue
		}
		pos := (sec - t0) / stepSec
		if pos < 0 {
			continue
		}
		if pos >= nBuckets {
			pos = nBuckets - 1
		}
		a := &accs[pos]
		if p.T > a.lastT {
			a.lastT = p.T
			a.gcNumLast = p.GCNum
			a.uptimeLast = p.Uptime
		}
		a.cpuSum += p.CPU
		a.cpuN++
		a.memSum += p.MemSys
		a.memN++
		a.heapSum += p.HeapAlloc
		a.heapN++
		a.sysSum += p.SysMem
		a.sysN++
		a.gorSum += float64(p.Goroutines)
		a.gorN++
		if p.GCPauseMs != nil {
			a.pauseSum += *p.GCPauseMs
			a.pauseN++
		}
		a.diskSum += p.Disk
		a.diskN++
		if p.Load1 != nil {
			a.loadSum += *p.Load1
			a.loadN++
		}
	}

	f64 := func(v float64) *float64 { return &v }
	buckets := make([]Bucket, 0, nBuckets)
	for i := range accs {
		a := &accs[i]
		if a.lastT == 0 {
			continue // 空桶
		}
		b := Bucket{
			Timestamp: util.JSONTime(time.Unix(t0+int64(i)*stepSec, 0)),
		}
		if a.cpuN > 0 {
			b.CPU = f64(round2(a.cpuSum / a.cpuN))
		}
		if a.memN > 0 {
			b.MemSys = f64(round2(a.memSum / a.memN))
		}
		if a.heapN > 0 {
			b.HeapAlloc = f64(round2(a.heapSum / a.heapN))
		}
		if a.sysN > 0 {
			b.SysMem = f64(round2(a.sysSum / a.sysN))
		}
		if a.gorN > 0 {
			b.Goroutines = f64(round2(a.gorSum / a.gorN))
		}
		if a.pauseN > 0 {
			b.GCPauseMs = f64(round2(a.pauseSum / a.pauseN))
		}
		if a.diskN > 0 {
			b.Disk = f64(round2(a.diskSum / a.diskN))
		}
		if a.loadN > 0 {
			b.Load1 = f64(round2(a.loadSum / a.loadN))
		}
		gc := a.gcNumLast
		up := a.uptimeLast
		b.GCNum = &gc
		b.Uptime = &up
		buckets = append(buckets, b)
	}

	return &Snapshot{
		WindowSeconds: windowSec,
		StepSeconds:   stepSec,
		Buckets:       buckets,
	}, nil
}
