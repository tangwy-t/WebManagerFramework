package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"time"
)

const (
	// AccessPrefix is the Redis key prefix for access token → userID mappings.
	AccessPrefix = "access:"
	// RefreshPrefix is the Redis key prefix for refresh token → userID mappings (keyed by userID).
	RefreshPrefix = "refresh:"
	// PermsPrefix is the Redis key prefix for cached user permissions (keyed by userID).
	PermsPrefix = "perms:"
)

// SessionStore is the shared Redis-backed implementation of the token and
// permission store contracts. NewSession returns it; consumers accept it
// through their own focused consumer-side interfaces (TokenStoreInterface,
// PermissionStoreInterface, SessionStoreInterface).
type SessionStore struct {
	cacheStore CacheStoreInterface
}

func NewSession(cacheStore CacheStoreInterface) *SessionStore {
	return &SessionStore{cacheStore: cacheStore}
}

func (s *SessionStore) StoreAccess(ctx context.Context, token string, userID uint64, ttl time.Duration) error {
	return s.cacheStore.Set(ctx, AccessPrefix+token, cast.ToString(userID), ttl)
}

func (s *SessionStore) StoreRefresh(ctx context.Context, userID uint64, token string, ttl time.Duration) error {
	return s.cacheStore.Set(ctx, fmt.Sprintf("%s%d", RefreshPrefix, userID), token, ttl)
}

func (s *SessionStore) StorePerms(ctx context.Context, userID uint64, perms []string, ttl time.Duration) error {
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	return s.cacheStore.Set(ctx, fmt.Sprintf("%s%d", PermsPrefix, userID), string(data), ttl)
}

func (s *SessionStore) RevokeAll(ctx context.Context, userID uint64, token string) error {
	// Split into individual Del calls to avoid CROSSSLOT error in Redis cluster mode
	// (keys with different prefixes hash to different slots).
	if err := s.cacheStore.Del(ctx, AccessPrefix+token); err != nil {
		return err
	}
	if err := s.cacheStore.Del(ctx, fmt.Sprintf("%s%d", RefreshPrefix, userID)); err != nil {
		return err
	}
	return s.cacheStore.Del(ctx, fmt.Sprintf("%s%d", PermsPrefix, userID))
}

func (s *SessionStore) RevokePerms(ctx context.Context, userID uint64) error {
	return s.cacheStore.Del(ctx, fmt.Sprintf("%s%d", PermsPrefix, userID))
}

// maxPermsInvalidation 一次全量失效最多删除的 perms 缓存键数(防御性上限)。
const maxPermsInvalidation = 100000

// RevokeAllPerms 失效所有用户的权限缓存。用于菜单权限点变更:影响面是
// "引用该菜单的全部角色 → 全部用户",逐角色追踪成本高且易漏;菜单管理为
// 低频管理操作,全清一次让所有用户下次请求时回源重建,最简且安全。
func (s *SessionStore) RevokeAllPerms(ctx context.Context) error {
	_, err := s.cacheStore.DeleteByPattern(ctx, PermsPrefix+"*", maxPermsInvalidation)
	return err
}

// IsAccessValid 校验 access token 是否在会话白名单中。
// 空串 = key 不存在 = 已登出/吊销/过期 → 无效。曾经依赖 goredis.Nil
// 判 miss,但 CacheStoreInterface.Get 的实现已把 Nil 翻译成空串,
// 死分支导致 miss 返回 (true, nil)——吊销校验整体 fail-open。
func (s *SessionStore) IsAccessValid(ctx context.Context, token string) (bool, error) {
	val, err := s.cacheStore.Get(ctx, AccessPrefix+token)
	if err != nil {
		return false, err
	}
	return val != "", nil
}

func (s *SessionStore) GetRefresh(ctx context.Context, userID uint64) (string, error) {
	val, err := s.cacheStore.Get(ctx, fmt.Sprintf("%s%d", RefreshPrefix, userID))
	if err != nil {
		return "", err
	}
	return val, nil // 空串 = 无有效 refresh token
}

func (s *SessionStore) DeleteRefresh(ctx context.Context, userID uint64) error {
	return s.cacheStore.Del(ctx, fmt.Sprintf("%s%d", RefreshPrefix, userID))
}

func (s *SessionStore) LoadPerms(ctx context.Context, userID uint64) ([]string, error) {
	cached, err := s.cacheStore.Get(ctx, fmt.Sprintf("%s%d", PermsPrefix, userID))
	if err != nil {
		return nil, err
	}
	if cached == "" {
		return nil, nil // 缓存 miss,消费方回源 DB
	}
	var perms []string
	if err := json.Unmarshal([]byte(cached), &perms); err != nil {
		return nil, err
	}
	return perms, nil
}
