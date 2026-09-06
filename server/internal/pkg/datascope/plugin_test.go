package datascope

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope/rule"
)

// scopeTestUser mirrors entity.SysUser's scope rules without importing the
// entity package (which would create an import cycle in tests).
type scopeTestUser struct {
	ID       uint64  `gorm:"column:id;primaryKey;autoIncrement:false"`
	Username string  `gorm:"column:username"`
	DeptID   *uint64 `gorm:"column:dept_id"`
}

func (scopeTestUser) TableName() string { return "sys_user" }

// DataScopeRules implements rule.DataScopeable — same shape as entity.SysUser.
func (scopeTestUser) DataScopeRules() []rule.ScopeRule {
	return []rule.ScopeRule{
		{Column: "dept_id", DimensionType: "dept"},
		{Column: "id", DimensionType: "self"},
	}
}

func newScopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&scopeTestUser{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	dept := uint64(10)
	rows := []scopeTestUser{
		{ID: 1, Username: "victim-other-dept", DeptID: &dept},
		{ID: 2, Username: "victim-same-dept", DeptID: &dept},
		{ID: 7, Username: "self", DeptID: &dept},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return db
}

func selfScopeCtx(userID uint64) context.Context {
	return WithScopeContext(context.Background(), &ScopeContext{
		UserID: userID,
		Dimensions: map[string]*ResolvedDimension{
			"self": {Level: ScopeSelf, SelfID: userID, AllowedIDs: []uint64{userID}},
		},
	})
}

// TestScopeCallbackBlocksOutOfScopeDelete proves the Delete callback injects
// scope conditions: a self-scoped user must not delete another user's row.
func TestScopeCallbackBlocksOutOfScopeDelete(t *testing.T) {
	db := newScopeTestDB(t)
	plugin := NewScopePlugin()
	plugin.RegisterPlugin(db)
	plugin.RegisterEntity("sys_user", scopeTestUser{})

	res := db.WithContext(selfScopeCtx(7)).Delete(&scopeTestUser{}, 1)
	if res.Error != nil {
		t.Fatalf("delete: %v", res.Error)
	}
	if res.RowsAffected != 0 {
		t.Fatalf("self-scoped delete of another user's row affected %d rows, want 0 (IDOR)", res.RowsAffected)
	}

	// Deleting own row must still work.
	res = db.WithContext(selfScopeCtx(7)).Delete(&scopeTestUser{}, 7)
	if res.Error != nil {
		t.Fatalf("delete own row: %v", res.Error)
	}
	if res.RowsAffected != 1 {
		t.Fatalf("self-scoped delete of own row affected %d rows, want 1", res.RowsAffected)
	}
}

// TestScopeCallbackBlocksOutOfScopeUpdate proves the Update callback injects
// scope conditions: a self-scoped user must not modify another user's row.
func TestScopeCallbackBlocksOutOfScopeUpdate(t *testing.T) {
	db := newScopeTestDB(t)
	plugin := NewScopePlugin()
	plugin.RegisterPlugin(db)
	plugin.RegisterEntity("sys_user", scopeTestUser{})

	res := db.WithContext(selfScopeCtx(7)).
		Model(&scopeTestUser{}).
		Where("id = ?", 2).
		Updates(map[string]any{"username": "hijacked"})
	if res.Error != nil {
		t.Fatalf("update: %v", res.Error)
	}
	if res.RowsAffected != 0 {
		t.Fatalf("self-scoped update of another user's row affected %d rows, want 0 (IDOR)", res.RowsAffected)
	}

	var name string
	db.WithContext(context.Background()).Model(&scopeTestUser{}).Where("id = ?", 2).Pluck("username", &name)
	if name != "victim-same-dept" {
		t.Fatalf("out-of-scope row was modified: username = %q", name)
	}
}

// TestScopeCallbackAllowsInScopeQuery guards the original Query behavior.
func TestScopeCallbackAllowsInScopeQuery(t *testing.T) {
	db := newScopeTestDB(t)
	plugin := NewScopePlugin()
	plugin.RegisterPlugin(db)
	plugin.RegisterEntity("sys_user", scopeTestUser{})

	var users []scopeTestUser
	if err := db.WithContext(selfScopeCtx(7)).Find(&users).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(users) != 1 || users[0].ID != 7 {
		t.Fatalf("self-scoped query returned %d rows, want exactly own row", len(users))
	}
}

// TestScopeCallbackNoContextIsPassthrough ensures statements without a scope
// context (system/background paths) are untouched.
func TestScopeCallbackNoContextIsPassthrough(t *testing.T) {
	db := newScopeTestDB(t)
	plugin := NewScopePlugin()
	plugin.RegisterPlugin(db)
	plugin.RegisterEntity("sys_user", scopeTestUser{})

	res := db.WithContext(context.Background()).Delete(&scopeTestUser{}, 1)
	if res.Error != nil {
		t.Fatalf("delete: %v", res.Error)
	}
	if res.RowsAffected != 1 {
		t.Fatalf("context-free delete affected %d rows, want 1", res.RowsAffected)
	}
}
