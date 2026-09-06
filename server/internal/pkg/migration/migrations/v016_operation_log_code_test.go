package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

// newOptLogCodeTestDB builds an in-memory sqlite DB with the dict and
// operation-log tables (v015 的 newTestDB 只建字典表,此处补操作日志表)。
func newOptLogCodeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.SysDictType{}, &entity.SysDictData{}, &entity.SysOperationLog{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// seedLegacyOptLogs 造出旧"status 时代"的数据:失败行 error_msg 非空、code 仍为默认 0。
func seedLegacyOptLogs(t *testing.T, db *gorm.DB) {
	t.Helper()
	msg40000 := `{"code":40000,"msg":"参数错误","data":null}`
	rows := []entity.SysOperationLog{
		{ID: 1, UserID: 1, Username: "admin", Module: "用户管理", OperationType: "修改", ErrorMsg: &msg40000},
		{ID: 2, UserID: 1, Username: "admin", Module: "用户管理", OperationType: "新增", ErrorMsg: ptr.To("panic: boom")},
		{ID: 3, UserID: 1, Username: "admin", Module: "用户管理", OperationType: "删除"},
	}
	for _, r := range rows {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("seed operation log: %v", err)
		}
	}
}

func TestV16OptResultCodeMigration(t *testing.T) {
	db := newOptLogCodeTestDB(t)
	seedLegacyOptLogs(t, db)

	if err := migrateOptResultCodeV16(db); err != nil {
		t.Fatalf("migration run 1: %v", err)
	}
	if err := migrateOptResultCodeV16(db); err != nil {
		t.Fatalf("migration run 2 (idempotency): %v", err)
	}

	// 1. 字典类型存在
	var dt entity.SysDictType
	if err := db.Where("code = ?", "sys_opt_result_code").First(&dt).Error; err != nil {
		t.Fatalf("sys_opt_result_code type not found: %v", err)
	}

	// 2. 全部 13 个业务码都已定义,list_class 正确
	var items []entity.SysDictData
	if err := db.Where("type_id = ?", dt.ID).Order("sort").Find(&items).Error; err != nil {
		t.Fatalf("query items: %v", err)
	}
	if len(items) != 13 {
		t.Fatalf("items = %d, want 13", len(items))
	}
	want := map[string]string{
		"0": "success", "10001": "warning", "10002": "warning", "10003": "warning",
		"10004": "warning", "10005": "danger", "10006": "warning", "10007": "danger",
		"10008": "danger", "40000": "danger", "40400": "danger", "40900": "danger",
		"50000": "danger",
	}
	for _, item := range items {
		if item.ListClass == nil || *item.ListClass != want[item.Value] {
			t.Fatalf("item %s: listClass = %v, want %s", item.Value, item.ListClass, want[item.Value])
		}
	}

	// 3. 历史失败行回填:信封 code 解析成功;解析失败兜底 50000;成功行保持 0
	cases := []struct {
		id   uint64
		want int
	}{
		{1, 40000},
		{2, apperror.CodeInternal},
		{3, apperror.CodeOK},
	}
	for _, c := range cases {
		var row entity.SysOperationLog
		if err := db.First(&row, c.id).Error; err != nil {
			t.Fatalf("query row %d: %v", c.id, err)
		}
		if row.Code != c.want {
			t.Fatalf("row %d: code = %d, want %d", c.id, row.Code, c.want)
		}
	}
}