package service

import (
	"context"
	"testing"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/dto/response"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"github.com/tangwy-t/webmanager-server/internal/pkg/session"
)

// ── fakes ────────────────────────────────────────────────────────────────

type fakeOnlineStore struct {
	uids  []uint64
	toks  map[uint64][]string
	metas map[string]*session.SessionMeta
	valid map[string]bool
}

func (f *fakeOnlineStore) ListOnlineUserIDs(context.Context, int64) ([]uint64, error) {
	return f.uids, nil
}
func (f *fakeOnlineStore) ListUserTokens(_ context.Context, uid uint64) ([]string, error) {
	return f.toks[uid], nil
}
func (f *fakeOnlineStore) BulkLoadSessionMeta(_ context.Context, tokens []string) (map[string]*session.SessionMeta, error) {
	out := map[string]*session.SessionMeta{}
	for _, t := range tokens {
		if m, ok := f.metas[t]; ok {
			out[t] = m
		}
	}
	return out, nil
}
func (f *fakeOnlineStore) IsAccessValid(_ context.Context, token string) (bool, error) {
	v, ok := f.valid[token]
	if !ok {
		return true, nil // 默认有效
	}
	return v, nil
}
func (f *fakeOnlineStore) RevokeOne(_ context.Context, uid uint64, token string) (bool, error) {
	return true, nil
}
func (f *fakeOnlineStore) DeleteRefresh(context.Context, uint64) error { return nil }

type fakeOnlineUsers struct {
	users []entity.SysUser
}

func (f *fakeOnlineUsers) FindByIDs(context.Context, []uint64) ([]entity.SysUser, error) {
	return f.users, nil
}

type fakeOnlineCfg struct{}

func (fakeOnlineCfg) GetString(_ context.Context, _ string, d string) string { return d }

func newOnlineSvc(store *fakeOnlineStore, users *fakeOnlineUsers) *OnlineUserService {
	return NewOnlineUserService(store, users, fakeOnlineCfg{}, logger.NewNop())
}

func deptPtr() *entity.SysDept { return &entity.SysDept{Name: "研发部"} }

// ── tests ────────────────────────────────────────────────────────────────

func TestListAggregatesSameDevice(t *testing.T) {
	store := &fakeOnlineStore{
		uids: []uint64{1},
		toks: map[uint64][]string{1: {"tokA", "tokB"}},
		metas: map[string]*session.SessionMeta{
			"tokA": {UserID: 1, IP: "1.1.1.1", UserAgent: "ua", Browser: "Chrome", OS: "Windows", LoginAt: 100, ExpireAt: 4102444800},
			"tokB": {UserID: 1, IP: "1.1.1.1", UserAgent: "ua", Browser: "Chrome", OS: "Windows", LoginAt: 100, ExpireAt: 4102444800},
		},
	}
	users := &fakeOnlineUsers{users: []entity.SysUser{{BaseEntity: entity.BaseEntity{ID: 1}, Username: "alice", Dept: deptPtr()}}}
	svc := newOnlineSvc(store, users)

	page, err := svc.List(context.Background(), &request.OnlineUserQuery{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	list := page.List.([]response.OnlineSession)
	if page.Total != 1 {
		t.Fatalf("聚合后应 1 行, got total=%d", page.Total)
	}
	if list[0].TokenCount != 2 || list[0].Username != "alice" {
		t.Fatalf("row = %+v", list[0])
	}
}

func TestListSkipsInvisibleUser(t *testing.T) {
	store := &fakeOnlineStore{
		uids: []uint64{1, 2},
		toks: map[uint64][]string{
			1: {"a"},
			2: {"b"},
		},
		metas: map[string]*session.SessionMeta{
			"a": {UserID: 1, IP: "1.1.1.1", UserAgent: "ua", Browser: "Chrome", OS: "Windows", LoginAt: 100, ExpireAt: 4102444800},
		},
	}
	users := &fakeOnlineUsers{users: []entity.SysUser{{BaseEntity: entity.BaseEntity{ID: 1}, Username: "alice"}}}
	svc := newOnlineSvc(store, users)
	page, _ := svc.List(context.Background(), &request.OnlineUserQuery{})
	if page.Total != 1 {
		t.Fatalf("不可见用户应被过滤, got total=%d", page.Total)
	}
}

func TestKickForbiddenWhenOutOfScope(t *testing.T) {
	store := &fakeOnlineStore{uids: []uint64{1}, toks: map[uint64][]string{1: {"tok"}}, metas: map[string]*session.SessionMeta{"tok": {UserID: 1}}}
	users := &fakeOnlineUsers{users: nil} // 可见用户为空 → 越权
	svc := newOnlineSvc(store, users)
	err := svc.Kick(context.Background(), &request.KickSessionReq{UserID: 1, Sid: session.SID("tok")}, "")
	if err == nil {
		t.Fatal("越权踢下线应报错")
	}
}

func TestKickSelfRejected(t *testing.T) {
	store := &fakeOnlineStore{toks: map[uint64][]string{}}
	users := &fakeOnlineUsers{users: []entity.SysUser{{BaseEntity: entity.BaseEntity{ID: 1}}}}
	svc := newOnlineSvc(store, users)
	tok := "mytoken"
	err := svc.Kick(context.Background(), &request.KickSessionReq{UserID: 1, Sid: session.SID(tok)}, tok)
	if err == nil {
		t.Fatal("自踢应被拒绝")
	}
}

func TestKickNotFoundWhenGroupMissing(t *testing.T) {
	store := &fakeOnlineStore{uids: []uint64{1}, toks: map[uint64][]string{1: {"tok"}}}
	users := &fakeOnlineUsers{users: []entity.SysUser{{BaseEntity: entity.BaseEntity{ID: 1}}}}
	svc := newOnlineSvc(store, users)
	err := svc.Kick(context.Background(), &request.KickSessionReq{UserID: 1, Sid: session.SID("nonexistent")}, "")
	if err == nil {
		t.Fatal("组不存在应报错")
	}
}

func TestKickRevokesGroup(t *testing.T) {
	tok := "tok"
	store := &fakeOnlineStore{
		uids:  []uint64{1},
		toks:  map[uint64][]string{1: {tok}},
		metas: map[string]*session.SessionMeta{tok: {UserID: 1}},
		valid: map[string]bool{tok: true},
	}
	users := &fakeOnlineUsers{users: []entity.SysUser{{BaseEntity: entity.BaseEntity{ID: 1}}}}
	svc := newOnlineSvc(store, users)
	if err := svc.Kick(context.Background(), &request.KickSessionReq{UserID: 1, Sid: session.SID(tok)}, ""); err != nil {
		t.Fatalf("kick: %v", err)
	}
}
