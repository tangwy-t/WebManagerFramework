package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

// newLoginLogCodeTestDB builds an in-memory sqlite DB containing sys_login_log
// (with the new code column) plus dict tables for the label update.
func newLoginLogCodeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:v017loginlog?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.SysDictType{}, &entity.SysDictData{}, &entity.SysLoginLog{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	// 模拟生产旧表:status 列在升级库中仍然存在(AutoMigrate 不删列),
	// 全新 sqlite 表按实体建表没有该列,手动补上。
	if err := db.Exec("ALTER TABLE sys_login_log ADD COLUMN status tinyint NOT NULL DEFAULT 1").Error; err != nil {
		t.Fatalf("add legacy status column: %v", err)
	}

	// 字典类型 + 10001 数据项(迁移把 label 改为“认证失败”)
	dt := entity.SysDictType{BaseEntity: entity.BaseEntity{ID: 220}, Code: "sys_opt_result_code", Name: "操作日志结果码", Status: ptr.To[int8](entity.DictTypeStatusEnabled)}
	if err := db.Create(&dt).Error; err != nil {
		t.Fatalf("seed dict type: %v", err)
	}
	dd := entity.SysDictData{BaseEntity: entity.BaseEntity{ID: 346}, TypeID: 220, Label: "未登录/令牌缺失", Value: "10001", Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)}
	if err := db.Create(&dd).Error; err != nil {
		t.Fatalf("seed dict data: %v", err)
	}

	// 旧 status 时代的登录日志(通过原生 SQL 插入,实体已无 status 字段)
	rows := []string{
		"(1, 'good', 1, '', 0, '2026-09-04 00:00:00')",   // 成功行 → code 保持 0
		"(2, 'bad1', 0, '用户名或密码错误', 0, '2026-09-04 00:00:01')", // → 10001
		"(3, 'bad2', 0, '账号已锁定', 0, '2026-09-04 00:00:02')",      // → 10008
		"(4, 'bad3', 0, '验证码错误', 0, '2026-09-04 00:00:03')",      // → 10005
		"(5, 'bad4', 0, '未知原因', 0, '2026-09-04 00:00:04')",        // → 兜底 10001
	}
	for _, r := range rows {
		if err := db.Exec("INSERT INTO sys_login_log (id, username, status, msg, code, login_time) VALUES " + r).Error; err != nil {
			t.Fatalf("seed login log %s: %v", r, err)
		}
	}
	return db
}

func TestV17LoginLogCodeMigration(t *testing.T) {
	db := newLoginLogCodeTestDB(t)

	if err := migrateLoginLogCodeV17(db); err != nil {
		t.Fatalf("migration run 1: %v", err)
	}
	if err := migrateLoginLogCodeV17(db); err != nil {
		t.Fatalf("migration run 2 (idempotency): %v", err)
	}

	// 1. 字典 10001 label 已改为“认证失败”
	var dd entity.SysDictData
	if err := db.Where("type_id = 220 AND value = '10001'").First(&dd).Error; err != nil {
		t.Fatalf("dict item 10001: %v", err)
	}
	if dd.Label != "认证失败" {
		t.Fatalf("label = %q, want 认证失败", dd.Label)
	}

	// 2. 逐行回填结果
	cases := []struct {
		id   uint64
		want int
	}{
		{1, apperror.CodeOK},
		{2, apperror.CodeUnauthorized},
		{3, apperror.CodeAccountLocked},
		{4, apperror.CodeCaptchaIncorrect},
		{5, apperror.CodeUnauthorized},
	}
	for _, c := range cases {
		var code int
		if err := db.Table("sys_login_log").Select("code").Where("id = ?", c.id).Scan(&code).Error; err != nil {
			t.Fatalf("query row %d: %v", c.id, err)
		}
		if code != c.want {
			t.Fatalf("row %d: code = %d, want %d", c.id, code, c.want)
		}
	}
}