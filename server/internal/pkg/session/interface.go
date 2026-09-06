package session

import (
	"context"
	"time"
)

type CacheStoreInterface interface {
	// Get 返回 key 对应的值;key 不存在时返回 ("", nil),不返回错误。
	// miss 语义由实现方保证(RedisStore 将 goredis.Nil 翻译为空串),
	// 消费方以空串判定 miss,不得假设会收到 goredis.Nil。
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	// DeleteByPattern 按 glob 模式批量删除 key(受 maxCount 上限保护),
	// 由 RedisStore 实现;用于低频管理操作的全量缓存失效。
	DeleteByPattern(ctx context.Context, pattern string, maxCount int64) (int64, error)
}
