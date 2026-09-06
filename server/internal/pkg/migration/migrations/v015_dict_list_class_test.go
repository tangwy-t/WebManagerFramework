package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.SysDictType{}, &entity.SysDictData{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedV7SampleData(t *testing.T, db *gorm.DB) {
	t.Helper()
	types := []entity.SysDictType{
		{BaseEntity: entity.BaseEntity{ID: 201}, Code: "sys_normal_disable", Name: "系统开关", Status: ptr.To[int8](1)},
		{BaseEntity: entity.BaseEntity{ID: 214}, Code: "sys_dict_status", Name: "字典数据状态", Status: ptr.To[int8](1)},
	}
	for _, dt := range types {
		if err := db.Create(&dt).Error; err != nil {
			t.Fatalf("seed type: %v", err)
		}
	}
	data := []entity.SysDictData{
		{BaseEntity: entity.BaseEntity{ID: 301}, TypeID: 201, Label: "正常", Value: "1", Sort: ptr.To(1)},
		{BaseEntity: entity.BaseEntity{ID: 302}, TypeID: 201, Label: "停用", Value: "0", Sort: ptr.To(2)},
		{BaseEntity: entity.BaseEntity{ID: 330}, TypeID: 214, Label: "启用", Value: "1", Sort: ptr.To(1)},
		{BaseEntity: entity.BaseEntity{ID: 331}, TypeID: 214, Label: "禁用", Value: "0", Sort: ptr.To(2)},
	}
	for _, dd := range data {
		if err := db.Create(&dd).Error; err != nil {
			t.Fatalf("seed data: %v", err)
		}
	}
}

func TestV15DictListClassMigration(t *testing.T) {
	db := newTestDB(t)
	seedV7SampleData(t, db)

	if err := seedDictListClassV15(db); err != nil {
		t.Fatalf("migration run 1: %v", err)
	}
	if err := seedDictListClassV15(db); err != nil {
		t.Fatalf("migration run 2 (idempotency): %v", err)
	}

	// 1. 新字典类型存在
	var dt entity.SysDictType
	if err := db.Where("code = ?", "sys_list_class").First(&dt).Error; err != nil {
		t.Fatalf("sys_list_class type not found: %v", err)
	}

	// 2. 5 个数据项存在且 list_class 自描述
	var items []entity.SysDictData
	if err := db.Where("type_id = ?", dt.ID).Order("sort").Find(&items).Error; err != nil {
		t.Fatalf("query items: %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("items = %d, want 5", len(items))
	}
	wantSelf := map[string]string{"primary": "primary", "success": "success", "info": "info", "warning": "warning", "danger": "danger"}
	for i, item := range items {
		if item.ListClass == nil || *item.ListClass != wantSelf[item.Value] {
			t.Fatalf("item[%d] %s: listClass = %v, want %s", i, item.Value, item.ListClass, wantSelf[item.Value])
		}
	}

	// 3. 现有项样式已分配（idempotency: 第二次运行后仍正确）
	cases := []struct {
		typeID      uint64
		value, want string
	}{
		{201, "1", "success"}, // 正常
		{201, "0", "danger"},  // 停用
		{214, "1", "success"}, // 启用
		{214, "0", "danger"},  // 禁用
	}
	for _, c := range cases {
		var got entity.SysDictData
		if err := db.Where("type_id = ? AND value = ?", c.typeID, c.value).First(&got).Error; err != nil {
			t.Fatalf("query %d/%s: %v", c.typeID, c.value, err)
		}
		if got.ListClass == nil || *got.ListClass != c.want {
			t.Fatalf("typeID=%d value=%s: listClass = %v, want %s", c.typeID, c.value, got.ListClass, c.want)
		}
	}

	// 4. 幂等：数据总量不变
	var count int64
	db.Model(&entity.SysDictData{}).Count(&count)
	if count != 9 { // 4 条预置 + 5 条新增
		t.Fatalf("total data count = %d, want 9", count)
	}
}
