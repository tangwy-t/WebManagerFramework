package repository

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/app"
	"github.com/tangwy-t/webmanager-server/internal/pkg/database"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"github.com/tangwy-t/webmanager-server/internal/pkg/snowflake"
	"gorm.io/gorm"
)

// newUserRoleTestDB 建 sqlite 内存库并迁移 user/role(含 many2many join 表)。
// 使用独立 DSN(带缓存共享关闭)避免与其他 :memory: 测试串库。
func newUserRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:userrole?mode=memory"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 用真实 snowflake node 注册回调,与生产一致:批量 INSERT join 行时自动生成 ID。
	node, err := snowflake.New(1, logger.NewNop())
	if err != nil {
		t.Fatalf("snowflake: %v", err)
	}
	database.NewCallbacks(node).Register(db)
	// production 中 SysUserRole 实体参与 MigrateAll,join 表才带 id 主键列;
	// 仅迁移 SysUser/SysRole 时 many2many 自动建的 join 表缺少 id。
	if err := db.AutoMigrate(&entity.SysUser{}, &entity.SysRole{}, &entity.SysUserRole{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedRole(t *testing.T, db *gorm.DB, id uint64, code string) {
	t.Helper()
	if err := db.Create(&entity.SysRole{
		BaseEntity: entity.BaseEntity{ID: id},
		Name:       code, Code: code,
	}).Error; err != nil {
		t.Fatalf("seed role %s: %v", code, err)
	}
}

func seedUser(t *testing.T, db *gorm.DB, id uint64, username string, roleIDs ...uint64) {
	t.Helper()
	user := &entity.SysUser{
		BaseEntity: entity.BaseEntity{ID: id},
		Username:   username,
		Password:   "hashed",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user %s: %v", username, err)
	}
	for _, rid := range roleIDs {
		if err := db.Model(user).Association("Roles").Append(
			&entity.SysRole{BaseEntity: entity.BaseEntity{ID: rid}}); err != nil {
			t.Fatalf("seed user-role join: %v", err)
		}
	}
}

func TestUserRepoFindPage_RoleIDFilter(t *testing.T) {
	db := newUserRoleTestDB(t)
	seedRole(t, db, 1, "admin")
	seedRole(t, db, 2, "dev")
	seedUser(t, db, 10, "alice", 1)
	seedUser(t, db, 11, "bob", 1, 2)
	seedUser(t, db, 12, "carol") // 无角色
	repo := NewUserRepository(db)
	ctx := context.Background()

	roleID := uint64(1)
	users, total, err := repo.FindPage(ctx, &request.UserQuery{
		PageRequest: app.PageRequest{Page: 1, PageSize: 10},
		RoleID:      &roleID,
	})
	if err != nil {
		t.Fatalf("FindPage: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	ids := map[uint64]bool{}
	for _, u := range users {
		ids[u.ID] = true
	}
	if !ids[10] || !ids[11] || ids[12] {
		t.Fatalf("unexpected role-filtered users: %v", ids)
	}

	// 无 RoleID 过滤时返回全部
	users2, total2, err := repo.FindPage(ctx, &request.UserQuery{
		PageRequest: app.PageRequest{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatalf("FindPage(all): %v", err)
	}
	if total2 != 3 || len(users2) != 3 {
		t.Fatalf("total(all) = %d, want 3", total2)
	}
}

func TestUserRepoAddUsersToRole_Idempotent(t *testing.T) {
	db := newUserRoleTestDB(t)
	seedRole(t, db, 1, "admin")
	seedRole(t, db, 2, "dev")
	seedUser(t, db, 10, "alice", 1)
	seedUser(t, db, 11, "bob")
	seedUser(t, db, 12, "carol")
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 第一次:alice 已在该角色,只应新增 bob
	if err := repo.AddUsersToRole(ctx, 1, []uint64{10, 11}); err != nil {
		t.Fatalf("AddUsersToRole#1: %v", err)
	}
	// 第二次:重复提交 alice+bob,另加 carol,不应产生重复行
	if err := repo.AddUsersToRole(ctx, 1, []uint64{10, 11, 12}); err != nil {
		t.Fatalf("AddUsersToRole#2: %v", err)
	}

	var count int64
	if err := db.Table("sys_user_role").Where("role_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("count join rows: %v", err)
	}
	if count != 3 { // alice、bob、carol 各一行
		t.Fatalf("join rows = %d, want 3", count)
	}
}

func TestUserRepoRemoveUsersFromRole_ScopedToRole(t *testing.T) {
	db := newUserRoleTestDB(t)
	seedRole(t, db, 1, "admin")
	seedRole(t, db, 2, "dev")
	seedUser(t, db, 10, "alice", 1, 2) // 同属两个角色
	seedUser(t, db, 11, "bob", 1)
	repo := NewUserRepository(db)
	ctx := context.Background()

	if err := repo.RemoveUsersFromRole(ctx, 1, []uint64{10}); err != nil {
		t.Fatalf("RemoveUsersFromRole: %v", err)
	}

	// alice 与角色1的关联被删除、与角色2的关联保留;bob 不受影响
	for _, tc := range []struct {
		userID, roleID uint64
		want           int64
	}{
		{10, 1, 0},
		{10, 2, 1},
		{11, 1, 1},
	} {
		var count int64
		if err := db.Table("sys_user_role").
			Where("user_id = ? AND role_id = ?", tc.userID, tc.roleID).
			Count(&count).Error; err != nil {
			t.Fatalf("count join rows: %v", err)
		}
		if count != tc.want {
			t.Fatalf("join(user=%d, role=%d) = %d, want %d", tc.userID, tc.roleID, count, tc.want)
		}
	}
}

func TestUserRepoFindRoleCodeByID(t *testing.T) {
	db := newUserRoleTestDB(t)
	seedRole(t, db, 1, "admin")
	repo := NewUserRepository(db)

	code, err := repo.FindRoleCodeByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindRoleCodeByID: %v", err)
	}
	if code != "admin" {
		t.Fatalf("code = %q, want admin", code)
	}
}
