package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
)

// newConfigNameTestDB builds an in-memory sqlite DB containing sys_config.
func newConfigNameTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:v018configname?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.SysConfig{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedConfigNameRow(t *testing.T, db *gorm.DB, id uint64, key, name string) {
	t.Helper()
	cfg := entity.SysConfig{
		BaseEntity:  entity.BaseEntity{ID: id},
		Name:        name,
		ConfigKey:   key,
		ConfigValue: "v",
		ConfigType:  "S",
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("seed config %s: %v", key, err)
	}
}

func TestV18ConfigNameBackfill(t *testing.T) {
	db := newConfigNameTestDB(t)

	// 内建配置项:名称为空 → 应回填
	seedConfigNameRow(t, db, 1, "sys.jwt.secret", "")
	// 内建配置项:已有名称 → 不应被覆盖(幂等)
	seedConfigNameRow(t, db, 2, "sys.jwt.accessExpire", "自定义名称")
	// 用户自建配置:名称为空 → 应保持原样
	seedConfigNameRow(t, db, 3, "my.custom.key", "")

	if err := backfillConfigNamesV18(db); err != nil {
		t.Fatalf("migration run 1: %v", err)
	}
	// 幂等:第二次运行不改变已回填的名称
	if err := backfillConfigNamesV18(db); err != nil {
		t.Fatalf("migration run 2 (idempotency): %v", err)
	}

	cases := []struct {
		key  string
		want string
	}{
		{"sys.jwt.secret", "JWT签名密钥"},
		{"sys.jwt.accessExpire", "自定义名称"},
		{"my.custom.key", ""},
	}
	for _, c := range cases {
		var got string
		if err := db.Table("sys_config").Select("name").Where("config_key = ?", c.key).Scan(&got).Error; err != nil {
			t.Fatalf("query %s: %v", c.key, err)
		}
		if got != c.want {
			t.Fatalf("name of %s = %q, want %q", c.key, got, c.want)
		}
	}
}