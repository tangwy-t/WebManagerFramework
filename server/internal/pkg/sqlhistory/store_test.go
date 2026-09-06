package sqlhistory

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// newTestStore 起一个内存 miniredis,返回 store、client 与清理函数。
func newTestStore(t *testing.T, ttl time.Duration) (*HistoryStore, *goredis.Client, func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	store := NewHistoryStore(rdb, ttl)
	cleanup := func() {
		_ = rdb.Close()
		mr.Close()
	}
	return store, rdb, cleanup
}

// storePt 构造一个采样点:T 为 now 前 offsetSec 秒,其余字段按入参。
func storePt(now time.Time, offsetSec, count int64, v float64) Point {
	return Point{
		T:     now.Add(-time.Duration(offsetSec) * time.Second).UnixMilli(),
		Count: count, AvgMs: v, P50Ms: v, P95Ms: v, P99Ms: v, MaxMs: v,
	}
}

// TestHistoryStoreAppendQuery 往返:Append 的点出现在 Query 结果里。
func TestHistoryStoreAppendQuery(t *testing.T) {
	store, _, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	now := time.Now()
	if err := store.Append(context.Background(), storePt(now, 0, 2, 10)); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := store.Append(context.Background(), storePt(now, 1, 3, 20)); err != nil {
		t.Fatalf("append: %v", err)
	}

	snap, err := store.Query(context.Background(), time.Minute, 3*time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	var total int64
	for _, b := range snap.Buckets {
		total += b.Count
	}
	if total != 5 {
		t.Fatalf("sum(count) = %d, want 5", total)
	}
	if len(snap.Buckets) == 0 {
		t.Fatal("buckets 不应为空")
	}
}

// TestHistoryStoreTrim 裁剪:第 28801 条写入后列表长度仍为 28800。
func TestHistoryStoreTrim(t *testing.T) {
	store, rdb, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	now := time.Now()
	for i := 0; i < maxPoints+5; i++ {
		if err := store.Append(context.Background(), storePt(now, 0, 1, 1)); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	ll, err := rdb.LLen(context.Background(), historyKey).Result()
	if err != nil {
		t.Fatalf("llen: %v", err)
	}
	if ll != maxPoints {
		t.Fatalf("llen = %d, want %d", ll, maxPoints)
	}
}

// TestHistoryStoreQueryTTL TTL 内共享同指针;不同 window/step 分键隔离。
func TestHistoryStoreQueryTTL(t *testing.T) {
	store, _, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	now := time.Now()
	_ = store.Append(context.Background(), storePt(now, 0, 1, 1))

	a, err := store.Query(context.Background(), time.Minute, 3*time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	b, err := store.Query(context.Background(), time.Minute, 3*time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if a != b {
		t.Fatal("TTL 内应返回同一份缓存快照")
	}
	c, err := store.Query(context.Background(), 5*time.Minute, 3*time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if a == c {
		t.Fatal("不同 window 的缓存项应相互独立")
	}
}

// TestHistoryStoreSkipsMalformed 坏 JSON 跳过,不影响有效点。
func TestHistoryStoreSkipsMalformed(t *testing.T) {
	store, rdb, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	// 先塞一条坏数据(列表头部),再 Append 一条有效点。
	if err := rdb.LPush(context.Background(), historyKey, "{bad json").Err(); err != nil {
		t.Fatalf("lpush: %v", err)
	}
	now := time.Now()
	if err := store.Append(context.Background(), storePt(now, 0, 1, 7)); err != nil {
		t.Fatalf("append: %v", err)
	}

	snap, err := store.Query(context.Background(), time.Minute, 3*time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	var total int64
	for _, b := range snap.Buckets {
		total += b.Count
	}
	if total != 1 {
		t.Fatalf("sum(count) = %d, want 1(坏点跳过)", total)
	}
}

// TestHistoryStoreReadsNewest 读头语义:只读最近 fetch 条。
// window=5s(step=1s)时 fetch=5/3*2+2=4(含时钟压缩安全系数);
// 取到的是最近 4 个点(offset 2/4/6/7),其中 6s/7s 点被 5s 窗口过滤
// → 剩 2 桶;实现若误读尾部,取到 offset 8/7/6/4,窗口内只剩 4s 点
// → 1 桶(≠2 仍可检出)。
func TestHistoryStoreReadsNewest(t *testing.T) {
	store, _, cleanup := newTestStore(t, time.Hour)
	defer cleanup()

	now := time.Now()
	for _, off := range []int64{8, 7, 6, 4, 2} { // 先旧后新,最新在头
		if err := store.Append(context.Background(), storePt(now, off, 1, 1)); err != nil {
			t.Fatalf("append %d: %v", off, err)
		}
	}

	snap, err := store.Query(context.Background(), 5*time.Second, time.Second)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(snap.Buckets) != 2 {
		t.Fatalf("buckets = %d, want 2(读头语义)", len(snap.Buckets))
	}
}
