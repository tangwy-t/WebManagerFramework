package serverstats

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// newTestStore 起一个内存 miniredis,返回 store 与清理函数。
func newTestStore(t *testing.T, ttl time.Duration) (*HistoryStore, func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	store := NewHistoryStore(rdb, ttl)
	cleanup := func() {
		_ = rdb.Close()
		mr.Close()
	}
	return store, cleanup
}

// appendAt 追加一个带给定时间偏移的点。
func appendAt(t *testing.T, s *HistoryStore, offset time.Duration, cpuValue float64) {
	t.Helper()
	if err := s.Append(context.Background(), Point{T: time.Now().Add(offset).UnixMilli(), CPU: cpuValue, Uptime: 100}); err != nil {
		t.Fatalf("append: %v", err)
	}
}

// 写入后可查询往返;window/step 透传;聚合结果包含两个点。
func TestHistoryStoreAppendQueryRoundtrip(t *testing.T) {
	store, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	appendAt(t, store, -5*time.Second, 10)
	appendAt(t, store, -1*time.Second, 30)

	snap, err := store.Query(context.Background(), 2*time.Minute, 2*time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if snap.WindowSeconds != 120 || snap.StepSeconds != 2 {
		t.Fatalf("window/step = %d/%d, want 120/2", snap.WindowSeconds, snap.StepSeconds)
	}
	total, cnt := 0.0, 0.0
	for _, b := range snap.Buckets {
		if b.CPU != nil {
			total += *b.CPU
			cnt++
		}
	}
	if cnt != 2 || total != 40 {
		t.Fatalf("cpu 桶值个数 = %v 合计 %v, want 2 个合计 40", cnt, total)
	}
}

// LTRIM 裁剪:写入超过 maxPoints 后列表长度封顶。
func TestHistoryStoreTrimCap(t *testing.T) {
	store, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	for i := 0; i < maxPoints+50; i++ {
		if err := store.Append(context.Background(), Point{T: time.Now().UnixMilli(), CPU: float64(i)}); err != nil {
			t.Fatalf("append #%d: %v", i, err)
		}
	}

	n, err := store.rdb.LLen(context.Background(), historyKey).Result()
	if err != nil {
		t.Fatalf("llen: %v", err)
	}
	if n != maxPoints {
		t.Fatalf("llen = %d, want %d", n, maxPoints)
	}
}

// TTL 内命中同一指针;过期后重新计算。
func TestHistoryStoreQueryTTL(t *testing.T) {
	store, cleanup := newTestStore(t, time.Hour)
	defer cleanup()
	appendAt(t, store, -1*time.Second, 10)

	a, err := store.Query(context.Background(), time.Minute, time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	b, err := store.Query(context.Background(), time.Minute, time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if a != b {
		t.Fatal("TTL 内应返回同一份缓存快照")
	}

	store2, cleanup2 := newTestStore(t, time.Nanosecond)
	defer cleanup2()
	appendAt(t, store2, -1*time.Second, 10)
	a2, err := store2.Query(context.Background(), time.Minute, time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	time.Sleep(time.Millisecond)
	b2, err := store2.Query(context.Background(), time.Minute, time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if a2 == b2 {
		t.Fatal("TTL 过期后应重新计算快照")
	}
}

// window/step 不同参数的缓存项互不串扰。
func TestHistoryStoreQueryKeyIsolation(t *testing.T) {
	store, cleanup := newTestStore(t, time.Hour)
	defer cleanup()
	appendAt(t, store, -1*time.Second, 10)

	a, err := store.Query(context.Background(), time.Minute, time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	b, err := store.Query(context.Background(), 5*time.Minute, time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if a == b {
		t.Fatal("不同 window 的缓存项应相互独立")
	}
}

// 单条坏数据跳过,不影响整体查询。
func TestHistoryStoreSkipsMalformedPoint(t *testing.T) {
	store, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	ctx := context.Background()
	if err := store.rdb.LPush(ctx, historyKey, "not-valid-json").Err(); err != nil {
		t.Fatalf("lpush: %v", err)
	}
	appendAt(t, store, -1*time.Second, 42)

	snap, err := store.Query(ctx, time.Minute, time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(snap.Buckets) != 1 {
		t.Fatalf("buckets = %d, want 1(坏数据被跳过)", len(snap.Buckets))
	}
	if snap.Buckets[0].CPU == nil || *snap.Buckets[0].CPU != 42 {
		t.Fatalf("cpu = %v, want 42", snap.Buckets[0].CPU)
	}
}

// 读头回归:列表头部是最新采样点(LPUSH 语义)。读尾会把窗口外的
// 最旧点取回来,window 小于列表跨度时整窗被截断为空(线上 BUG1)。
func TestHistoryStoreQueryReadsNewestPoints(t *testing.T) {
	store, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	// 30 分钟前的 50 个旧点:超出 1m 窗口;读尾(旧 BUG)时
	// LRange(-32,-1) 只取到最旧的 32 个点,窗口内整窗被截断为空。
	for i := 0; i < 50; i++ {
		appendAt(t, store, -30*time.Minute+time.Duration(i)*2*time.Second, 100+float64(i))
	}
	// 最新采样点位于头部,必须出现在查询结果里。
	appendAt(t, store, -1*time.Second, 10)

	snap, err := store.Query(context.Background(), time.Minute, 2*time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	var got []float64
	for _, b := range snap.Buckets {
		if b.CPU != nil {
			got = append(got, *b.CPU)
		}
	}
	if len(got) != 1 || got[0] != 10 {
		t.Fatalf("应只返回窗口内最新采样点 cpu=10, got %v", got)
	}
}
