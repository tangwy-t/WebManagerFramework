// Package datascope_test 是 datascope 插件的黑盒测试:使用真实 entity.SysUser
// (真实 DataScopeRules 与 TableName),不再手工镜像规则/表名 —— 此前的内包
// 测试与生产模型各自维护一份规则,实体侧漂移无法被测试察觉。
package datascope_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope"
)

func newScopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.SysUser{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	dept := uint64(10)
	rows := []entity.SysUser{
		{BaseEntity: entity.BaseEntity{ID: 1}, Username: "victim-other-dept", DeptID: &dept},
		{BaseEntity: entity.BaseEntity{ID: 2}, Username: "victim-same-dept", DeptID: &dept},
		{BaseEntity: entity.BaseEntity{ID: 7}, Username: "self", DeptID: &dept},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return db
}

func selfScopeCtx(userID uint64) context.Context {
	return datascope.WithScopeContext(context.Background(), &datascope.ScopeContext{
		UserID: userID,
		Dimensions: map[string]*datascope.ResolvedDimension{
			"self": {Level: datascope.ScopeSelf, SelfID: userID, AllowedIDs: []uint64{userID}},
		},
	})
}

// TestScopeCallbackBlocksOutOfScopeDelete Delete 回调注入 scope 条件:
// self 用户不得删除他人行(IDOR 防线),但可删除自己。
func TestScopeCallbackBlocksOutOfScopeDelete(t *testing.T) {
	db := newScopeTestDB(t)
	plugin := datascope.NewScopePlugin()
	plugin.RegisterPlugin(db)
	plugin.RegisterEntity("sys_user", entity.SysUser{})

	res := db.WithContext(selfScopeCtx(7)).Delete(&entity.SysUser{}, 1)
	if res.Error != nil {
		t.Fatalf("delete: %v", res.Error)
	}
	if res.RowsAffected != 0 {
		t.Fatalf("self-scoped delete of another user's row affected %d rows, want 0 (IDOR)", res.RowsAffected)
	}

	res = db.WithContext(selfScopeCtx(7)).Delete(&entity.SysUser{}, 7)
	if res.Error != nil {
		t.Fatalf("delete own row: %v", res.Error)
	}
	if res.RowsAffected != 1 {
		t.Fatalf("self-scoped delete of own row affected %d rows, want 1", res.RowsAffected)
	}
}

// TestScopeCallbackBlocksOutOfScopeUpdate Update 回调注入 scope 条件:
// self 用户不得修改他人行。
func TestScopeCallbackBlocksOutOfScopeUpdate(t *testing.T) {
	db := newScopeTestDB(t)
	plugin := datascope.NewScopePlugin()
	plugin.RegisterPlugin(db)
	plugin.RegisterEntity("sys_user", entity.SysUser{})

	res := db.WithContext(selfScopeCtx(7)).
		Model(&entity.SysUser{}).
		Where("id = ?", 2).
		Updates(map[string]any{"username": "hijacked"})
	if res.Error != nil {
		t.Fatalf("update: %v", res.Error)
	}
	if res.RowsAffected != 0 {
		t.Fatalf("self-scoped update of another user's row affected %d rows, want 0 (IDOR)", res.RowsAffected)
	}

	var name string
	db.WithContext(context.Background()).Model(&entity.SysUser{}).Where("id = ?", 2).Pluck("username", &name)
	if name != "victim-same-dept" {
		t.Fatalf("out-of-scope row was modified: username = %q", name)
	}
}

// TestScopeCallbackAllowsInScopeQuery 查询仍只返回自己范围内的行。
func TestScopeCallbackAllowsInScopeQuery(t *testing.T) {
	db := newScopeTestDB(t)
	plugin := datascope.NewScopePlugin()
	plugin.RegisterPlugin(db)
	plugin.RegisterEntity("sys_user", entity.SysUser{})

	var users []entity.SysUser
	if err := db.WithContext(selfScopeCtx(7)).Find(&users).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(users) != 1 || users[0].ID != 7 {
		t.Fatalf("self-scoped query returned %d rows, want exactly own row", len(users))
	}
}

// TestScopeCallbackNoContextIsPassthrough 无 scope ctx 的系统/后台路径不受影响。
func TestScopeCallbackNoContextIsPassthrough(t *testing.T) {
	db := newScopeTestDB(t)
	plugin := datascope.NewScopePlugin()
	plugin.RegisterPlugin(db)
	plugin.RegisterEntity("sys_user", entity.SysUser{})

	res := db.WithContext(context.Background()).Delete(&entity.SysUser{}, 1)
	if res.Error != nil {
		t.Fatalf("delete: %v", res.Error)
	}
	if res.RowsAffected != 1 {
		t.Fatalf("context-free delete affected %d rows, want 1", res.RowsAffected)
	}
}
