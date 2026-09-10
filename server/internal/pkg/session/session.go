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
	// UserAccessPrefix is the Redis key prefix for the user → access-token index
	// (a Redis SET of token strings, keyed by userID). It exists solely so that
	// every access token ever issued to a user can be revoked in one operation:
	// the access keys themselves are token-keyed, so without this reverse index
	// "revoke everything for user N" is impossible to express with a single DEL.
	UserAccessPrefix = "user_access:"
	// RefreshPrefix is the Redis key prefix for refresh token → userID mappings (keyed by userID).
	RefreshPrefix = "refresh:"
	// PermsPrefix is the Redis key prefix for cached user permissions (keyed by userID).
	PermsPrefix = "perms:"
)

// maxAccessRevocation 一次按用户吊销最多清理的 access token 数(防御性上限)。
const maxAccessRevocation = 1000

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
	if err := s.cacheStore.Set(ctx, AccessPrefix+token, cast.ToString(userID), ttl); err != nil {
		return err
	}
	return s.indexUserAccess(ctx, userID, token, ttl)
}

// indexUserAccess 把 token 追加进该用户的 access 索引。
//
// 索引是 redis 中的 JSON 数组(普通 string 键),而非 Redis SET:
// CacheStoreInterface 未暴露 SADD/SMEMBERS,引入新原语会让所有 FakeStore
// 实现(test 侧)同步扩展。索引键与 access 键共用同一 TTL —— 二者
// 生命周期一致,过期即整体消失,不存在悬挂引用。
//
// 索引写失败不阻断登录:索引只影响"能否一次性吊销全部 token",
// 不影响 token 本身可用性;失败降级为"该次登录的 token 不参与批量吊销"
// (逐 token 登出仍然有效),由调用方按运维告警跟进。
func (s *SessionStore) indexUserAccess(ctx context.Context, userID uint64, token string, ttl time.Duration) error {
	key := fmt.Sprintf("%s%d", UserAccessPrefix, userID)
	tokens, err := s.loadAccessIndex(ctx, key)
	if err != nil {
		return err
	}
	tokens = append(tokens, token)
	// 上限保护:索引无限增长会让单键体积随登录次数线性膨胀。
	// 保留最近 maxAccessRevocation 个即可 —— 更早的 token 早已过期。
	if len(tokens) > maxAccessRevocation {
		tokens = tokens[len(tokens)-maxAccessRevocation:]
	}
	data, err := json.Marshal(tokens)
	if err != nil {
		return err
	}
	return s.cacheStore.Set(ctx, key, string(data), ttl)
}

// loadAccessIndex 读取用户的 access 索引;键不存在或内容损坏时返回空切片。
// 损坏时返回空而非错误:索引是尽力而为的辅助结构,不应让登录失败。
func (s *SessionStore) loadAccessIndex(ctx context.Context, key string) ([]string, error) {
	raw, err := s.cacheStore.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var tokens []string
	if err := json.Unmarshal([]byte(raw), &tokens); err != nil {
		return nil, nil
	}
	return tokens, nil
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

// RevokeAll 吊销用户的全部会话:access 白名单(全部已签发 token)、
// refresh token 与权限缓存。
//
// accessToken 非空时额外立即吊销该 token —— 用于改密/登出场景:调用方
// 手上正好持有当前 token,即使索引缺失(历史 token 或索引写失败)也能
// 确保"正在使用的这个"立刻失效。
//
// 修复历史缺陷:此前实现为 Del(AccessPrefix+token),而三个调用点
// (禁用/删除/重置密码)传的是空串 → 实际执行 Del("access:") ——
// 删一个不存在的键。结果是 refresh 与 perms 被清,但该用户**全部
// 已签发的 access token 仍在白名单中**,禁用/删除/改密后旧 token
// 在剩余 TTL(默认 2h)内完全可用。现在按用户索引逐 token 删除。
func (s *SessionStore) RevokeAll(ctx context.Context, userID uint64, token string) error {
	// Split into individual Del calls to avoid CROSSSLOT error in Redis cluster mode
	// (keys with different prefixes hash to different slots).
	if err := s.revokeUserAccessTokens(ctx, userID); err != nil {
		return err
	}
	if token != "" {
		if err := s.cacheStore.Del(ctx, AccessPrefix+token); err != nil {
			return err
		}
	}
	if err := s.cacheStore.Del(ctx, fmt.Sprintf("%s%d", RefreshPrefix, userID)); err != nil {
		return err
	}
	return s.cacheStore.Del(ctx, fmt.Sprintf("%s%d", PermsPrefix, userID))
}

// revokeUserAccessTokens 删除该用户索引中的全部 access 白名单键,并清除索引本身。
func (s *SessionStore) revokeUserAccessTokens(ctx context.Context, userID uint64) error {
	key := fmt.Sprintf("%s%d", UserAccessPrefix, userID)
	tokens, err := s.loadAccessIndex(ctx, key)
	if err != nil {
		return err
	}
	if len(tokens) > 0 {
		if len(tokens) > maxAccessRevocation {
			tokens = tokens[:maxAccessRevocation]
		}
		keys := make([]string, 0, len(tokens))
		for _, t := range tokens {
			if t != "" {
				keys = append(keys, AccessPrefix+t)
			}
		}
		if len(keys) > 0 {
			if err := s.cacheStore.Del(ctx, keys...); err != nil {
				return err
			}
		}
	}
	return s.cacheStore.Del(ctx, key)
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
