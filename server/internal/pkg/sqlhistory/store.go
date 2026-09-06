package sqlhistory

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Store SQL 历史存储接口(消费方 service 定义)。
type Store interface {
	Append(ctx context.Context, p Point) error
	Query(ctx context.Context, window, step time.Duration) (*Snapshot, error)
}

// defaultQueryTTL 历史查询结果缓存有效期:窗口期内并发请求共享同一份
// 计算结果(单飞),与 serverstats.HistoryStore 一致。
const defaultQueryTTL = time.Second

// queryCacheMaxEntries 缓存键族上限(window × step 组合),超限整体重建。
const queryCacheMaxEntries = 4

type queryKey struct {
	window, step time.Duration
}

type queryEntry struct {
	snap *Snapshot
	at   time.Time
}

// HistoryStore SQL 历史的 Redis 滚动窗口实现。
// Append 写入采样点(LPUSH + LTRIM);Query 读头并内存聚合,
// 结果带短 TTL 缓存。返回的快照是共享只读对象,调用方不得修改。
type HistoryStore struct {
	rdb goredis.UniversalClient
	ttl time.Duration

	mu    sync.Mutex
	cache map[queryKey]queryEntry
}

// NewHistoryStore 创建 HistoryStore。
// 可选 ttl 参数仅供测试注入;生产使用默认 1 秒。
func NewHistoryStore(rdb goredis.UniversalClient, ttl ...time.Duration) *HistoryStore {
	t := defaultQueryTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		t = ttl[0]
	}
	return &HistoryStore{
		rdb:   rdb,
		ttl:   t,
		cache: make(map[queryKey]queryEntry),
	}
}

// Append 把一个采样点压入滚动窗口头部并裁剪到 maxPoints 条。
// 两条命令走 TxPipeline,一次往返。
func (s *HistoryStore) Append(ctx context.Context, p Point) error {
	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}
	pipe := s.rdb.TxPipeline()
	pipe.LPush(ctx, historyKey, payload)
	pipe.LTrim(ctx, historyKey, 0, maxPoints-1)
	_, err = pipe.Exec(ctx)
	return err
}

// Query 返回 [now-window, now] 内按 step 聚合的历史快照。
func (s *HistoryStore) Query(ctx context.Context, window, step time.Duration) (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	key := queryKey{window: window, step: step}
	if e, ok := s.cache[key]; ok && now.Sub(e.at) < s.ttl {
		return e.snap, nil
	}

	snap, err := s.queryUncached(ctx, window, step, now)
	if err != nil {
		return nil, err
	}
	if len(s.cache) >= queryCacheMaxEntries {
		s.cache = make(map[queryKey]queryEntry)
	}
	s.cache[key] = queryEntry{snap: snap, at: now}
	return snap, nil
}

// queryUncached 从 Redis 读头部采样点并聚合;单条坏数据跳过,不影响整体。
func (s *HistoryStore) queryUncached(ctx context.Context, window, step time.Duration, now time.Time) (*Snapshot, error) {
	// 取点数 = window/采样周期 × 2 安全系数 + 2(保证边界桶完整)。
	// 与服务器监控同一问题:点时间戳取自墙钟,虚拟机环境墙钟步进会
	// 压缩点的时间戳密度,只按 window/采样周期 取点会覆盖不满窗口。
	// 多取的点会被 aggregate 按 cutoff 丢弃,无副作用。
	fetch := int(window.Seconds())/sampleStepSeconds*2 + 2
	if fetch < 1 {
		fetch = 1
	}
	if fetch > maxPoints {
		fetch = maxPoints
	}

	// LPUSH 后最新点在列表头部(index 0),必须读头而不是尾;
	// 读尾会把窗口外最旧的点取回来,导致 window 较小时整窗被截断为空
	// (与服务器监控 BUG1 修复结论一致)。
	raw, err := s.rdb.LRange(ctx, historyKey, 0, int64(fetch-1)).Result()
	if err != nil {
		return nil, err
	}

	points := make([]Point, 0, len(raw))
	for _, item := range raw {
		var p Point
		if json.Unmarshal([]byte(item), &p) != nil {
			continue
		}
		points = append(points, p)
	}
	return aggregate(points, window, step, now)
}
