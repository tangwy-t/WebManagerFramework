package session

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeCache 是进程内 CacheStoreInterface 替身:Get 对缺失键返回 ("", nil),
// 与 RedisStore 的 miss 翻译语义一致。
type fakeCache struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string]string{}} }

func (f *fakeCache) Get(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data[key], nil
}

func (f *fakeCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = value
	return nil
}

func (f *fakeCache) Del(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range keys {
		delete(f.data, k)
	}
	return nil
}

func (f *fakeCache) DeleteByPattern(_ context.Context, pattern string, _ int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prefix := strings.TrimSuffix(pattern, "*")
	var n int64
	for k := range f.data {
		if strings.HasPrefix(k, prefix) {
			delete(f.data, k)
			n++
		}
	}
	return n, nil
}

// TestAccessWhitelistRoundTrip 会话白名单:存→有效→吊销→无效。
func TestAccessWhitelistRoundTrip(t *testing.T) {
	ctx := context.Background()
	c := newFakeCache()
	s := NewSession(c)

	if err := s.StoreAccess(ctx, "tok-1", 7, time.Minute); err != nil {
		t.Fatalf("StoreAccess: %v", err)
	}
	ok, err := s.IsAccessValid(ctx, "tok-1")
	if err != nil || !ok {
		t.Fatalf("IsAccessValid = %v/%v, want true/nil", ok, err)
	}
	// miss = 已登出/吊销/过期 → false
	ok, err = s.IsAccessValid(ctx, "tok-missing")
	if err != nil || ok {
		t.Fatalf("miss IsAccessValid = %v/%v, want false/nil(白名单缺失必须判无效)", ok, err)
	}
	if err := s.RevokeAll(ctx, 7, "tok-1"); err != nil {
		t.Fatalf("RevokeAll: %v", err)
	}
	ok, _ = s.IsAccessValid(ctx, "tok-1")
	if ok {
		t.Fatal("RevokeAll 后 token 仍有效")
	}
}

// TestPermsCacheRoundTrip 权限缓存:JSON 数组存取闭环 + 回源 miss 语义。
func TestPermsCacheRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewSession(newFakeCache())

	perms, err := s.LoadPerms(ctx, 7)
	if err != nil || perms != nil {
		t.Fatalf("miss LoadPerms = %v/%v, want nil/nil(消费方回源 DB)", perms, err)
	}
	if err := s.StorePerms(ctx, 7, []string{"a", "b"}, time.Minute); err != nil {
		t.Fatalf("StorePerms: %v", err)
	}
	perms, err = s.LoadPerms(ctx, 7)
	if err != nil || len(perms) != 2 || perms[1] != "b" {
		t.Fatalf("LoadPerms = %v/%v", perms, err)
	}
	if err := s.RevokePerms(ctx, 7); err != nil {
		t.Fatalf("RevokePerms: %v", err)
	}
	perms, _ = s.LoadPerms(ctx, 7)
	if perms != nil {
		t.Fatal("RevokePerms 后应回源 miss")
	}
}

// TestRefreshKeyRoundTrip refresh 键以 userID 为 key,覆盖存取/删除。
func TestRefreshKeyRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewSession(newFakeCache())

	if err := s.StoreRefresh(ctx, 7, "rt-1", time.Hour); err != nil {
		t.Fatalf("StoreRefresh: %v", err)
	}
	got, err := s.GetRefresh(ctx, 7)
	if err != nil || got != "rt-1" {
		t.Fatalf("GetRefresh = %q/%v", got, err)
	}
	if err := s.DeleteRefresh(ctx, 7); err != nil {
		t.Fatalf("DeleteRefresh: %v", err)
	}
	got, _ = s.GetRefresh(ctx, 7)
	if got != "" {
		t.Fatalf("删除后 GetRefresh = %q, want \"\"", got)
	}
}

// TestRevokeAllPermsPattern 全量失效按 perms:* 前缀清理。
func TestRevokeAllPermsPattern(t *testing.T) {
	ctx := context.Background()
	c := newFakeCache()
	s := NewSession(c)

	for _, uid := range []uint64{1, 2, 3} {
		if err := s.StorePerms(ctx, uid, []string{"x"}, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	_ = s.StoreAccess(ctx, "tok", 1, time.Minute) // 非 perms 键不可被误删
	if err := s.RevokeAllPerms(ctx); err != nil {
		t.Fatalf("RevokeAllPerms: %v", err)
	}
	if ok, _ := s.IsAccessValid(ctx, "tok"); !ok {
		t.Fatal("RevokeAllPerms 误删了 access 白名单键")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.data {
		if strings.HasPrefix(k, PermsPrefix) {
			t.Fatalf("RevokeAllPerms 后仍有 perms 键残留: %q", k)
		}
	}
}