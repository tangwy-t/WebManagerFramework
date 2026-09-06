package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

// v021ExpectedRows 新值域四行的预期文案、样式与默认项(与设计规格 §4 及线上手工修订一致)。
var v021ExpectedRows = map[string]struct {
	label, listClass string
	isDefault        int8
}{
	"0": {"全体成员", "primary", entity.DictDataDefaultYes},
	"1": {"指定角色", "warning", entity.DictDataDefaultNo},
	"2": {"指定部门", "success", entity.DictDataDefaultNo},
	"3": {"指定个人", "info", entity.DictDataDefaultNo},
}

// newPublishTypeTestDB builds an in-memory sqlite DB with dict tables only.
// name 区分每次调用的独立内存库(cache=shared 时同进程内同名 DSN 共享同一库)。
func newPublishTypeTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.SysDictType{}, &entity.SysDictData{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// seedV7PublishType 复刻 v007 种子里 sys_notice_publish_type 的旧值域(全新安装形态)。
func seedV7PublishType(t *testing.T, db *gorm.DB) {
	t.Helper()
	dt := entity.SysDictType{
		BaseEntity: entity.BaseEntity{ID: 207},
		Code:       "sys_notice_publish_type",
		Name:       "通知发布类型",
		Status:     ptr.To[int8](entity.DictTypeStatusEnabled),
	}
	if err := db.Create(&dt).Error; err != nil {
		t.Fatalf("seed type: %v", err)
	}
	rows := []entity.SysDictData{
		{BaseEntity: entity.BaseEntity{ID: 315}, TypeID: dt.ID, Label: "全体", Value: "1", Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 316}, TypeID: dt.ID, Label: "指定用户", Value: "2", Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
	}
	for _, r := range rows {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("seed data: %v", err)
		}
	}
}

// assertV21PublishType 断言类型名与 0-3 四行数据均正确(按 code 定位 type_id)。
func assertV21PublishType(t *testing.T, db *gorm.DB) {
	t.Helper()
	var dt entity.SysDictType
	if err := db.Where("code = ?", "sys_notice_publish_type").First(&dt).Error; err != nil {
		t.Fatalf("dict type not found: %v", err)
	}
	if dt.Name != "通知接收范围" {
		t.Fatalf("dict type name = %q, want 通知接收范围", dt.Name)
	}

	var items []entity.SysDictData
	if err := db.Where("type_id = ?", dt.ID).Order("sort").Find(&items).Error; err != nil {
		t.Fatalf("query items: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("items = %d, want 4(旧值域残留或新行缺失)", len(items))
	}
	for _, it := range items {
		want, ok := v021ExpectedRows[it.Value]
		if !ok {
			t.Fatalf("unexpected value %q(旧值域残留)", it.Value)
		}
		if it.Label != want.label {
			t.Fatalf("item %s: label = %q, want %q", it.Value, it.Label, want.label)
		}
		if it.ListClass == nil || *it.ListClass != want.listClass {
			t.Fatalf("item %s: listClass = %v, want %s", it.Value, it.ListClass, want.listClass)
		}
		if it.IsDefault == nil || *it.IsDefault != want.isDefault {
			t.Fatalf("item %s: isDefault = %v, want %d", it.Value, it.IsDefault, want.isDefault)
		}
	}
}

// TestV21NoticePublishTypeFix 全新安装:v007 旧值域 → 迁移修正为 0-3,重复执行幂等,
// 且旧两行被物理删除(Unscoped 查询数为 0)。
func TestV21NoticePublishTypeFix(t *testing.T) {
	db := newPublishTypeTestDB(t, "v021-fix")
	seedV7PublishType(t, db)

	if err := migrateNoticePublishTypeV21(db); err != nil {
		t.Fatalf("migration run 1: %v", err)
	}
	if err := migrateNoticePublishTypeV21(db); err != nil {
		t.Fatalf("migration run 2 (idempotency): %v", err)
	}

	assertV21PublishType(t, db)

	var legacy int64
	if err := db.Unscoped().Model(&entity.SysDictData{}).
		Where("type_id = (SELECT id FROM sys_dict_type WHERE code = 'sys_notice_publish_type' LIMIT 1) AND value IN ?",
			[]string{"全体", "指定用户"}).
		Count(&legacy).Error; err != nil {
		t.Fatalf("count legacy: %v", err)
	}
	if legacy != 0 {
		t.Fatalf("legacy rows by label = %d, want 0(旧两行应被物理删除)", legacy)
	}
}

// TestV21NoticePublishTypeAlreadyRevised 已按手工 SQL 修订过的库:守卫跳过数据替换,
// 运营后续改动(如改过文案)不被覆盖;类型名修正仍幂等执行。
func TestV21NoticePublishTypeAlreadyRevised(t *testing.T) {
	db := newPublishTypeTestDB(t, "v021-revised")
	seedV7PublishType(t, db)

	// 模拟手工 SQL 已应用:换成 0-3 四行,并把「全体成员」文案改成运营自定义值。
	var dt entity.SysDictType
	if err := db.Where("code = ?", "sys_notice_publish_type").First(&dt).Error; err != nil {
		t.Fatalf("dict type not found: %v", err)
	}
	if err := db.Unscoped().Where("type_id = ?", dt.ID).Delete(&entity.SysDictData{}).Error; err != nil {
		t.Fatalf("cleanup old rows: %v", err)
	}
	custom := []entity.SysDictData{
		{BaseEntity: entity.BaseEntity{ID: 358}, TypeID: dt.ID, Label: "所有人", Value: "0", ListClass: ptr.To("primary"), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 359}, TypeID: dt.ID, Label: "指定角色", Value: "1", ListClass: ptr.To("warning"), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 360}, TypeID: dt.ID, Label: "指定部门", Value: "2", ListClass: ptr.To("success"), Sort: ptr.To(3), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 361}, TypeID: dt.ID, Label: "指定个人", Value: "3", ListClass: ptr.To("info"), Sort: ptr.To(4), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
	}
	for _, r := range custom {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("seed revised row: %v", err)
		}
	}

	if err := migrateNoticePublishTypeV21(db); err != nil {
		t.Fatalf("migration on revised db: %v", err)
	}

	// 类型名修正幂等执行。
	if err := db.Where("code = ?", "sys_notice_publish_type").First(&dt).Error; err != nil {
		t.Fatalf("reload dict type: %v", err)
	}
	if dt.Name != "通知接收范围" {
		t.Fatalf("dict type name = %q, want 通知接收范围", dt.Name)
	}

	// 数据不被替换:「所有人」保留;行数仍为 4。
	var all entity.SysDictData
	if err := db.Where("type_id = ? AND value = '0'", dt.ID).First(&all).Error; err != nil {
		t.Fatalf("value 0 row missing: %v", err)
	}
	if all.Label != "所有人" {
		t.Fatalf("value 0 label = %q, want 所有人(守卫不应覆盖运营改动)", all.Label)
	}
	var count int64
	if err := db.Model(&entity.SysDictData{}).Where("type_id = ?", dt.ID).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 4 {
		t.Fatalf("rows = %d, want 4", count)
	}
}

// TestV21VersionRegistered 迁移注册进注册表(版本号唯一性由 Register 的 panic 校验兜底)。
func TestV21VersionRegistered(t *testing.T) {
	found := false
	for _, m := range migration.All() {
		if m.Version == 21 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("version 21 not registered; registered: %+v", migration.All())
	}
}