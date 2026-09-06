package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     10,
		Description: "写入 API 限流配置项 (sys.rateLimit.*)",
		Up:          seedRateLimitConfigV10,
	})
}

func seedRateLimitConfigV10(tx *gorm.DB) error {
	configs := []entity.SysConfig{
		{
			BaseEntity:  entity.BaseEntity{ID: 25},
			ConfigKey:   "sys.rateLimit.global.enabled",
			ConfigValue: "true",
			ConfigType:  "B",
			Remark:      ptr.To("全局 API 限流开关"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 26},
			ConfigKey:   "sys.rateLimit.global.limit",
			ConfigValue: "100",
			ConfigType:  "N",
			Remark:      ptr.To("全局每窗口最大请求数"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 27},
			ConfigKey:   "sys.rateLimit.global.windowSecs",
			ConfigValue: "60",
			ConfigType:  "N",
			Remark:      ptr.To("全局限流窗口大小(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 28},
			ConfigKey:   "sys.rateLimit.login.enabled",
			ConfigValue: "true",
			ConfigType:  "B",
			Remark:      ptr.To("登录接口限流开关"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 29},
			ConfigKey:   "sys.rateLimit.login.limit",
			ConfigValue: "10",
			ConfigType:  "N",
			Remark:      ptr.To("登录接口每窗口最大请求数"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 30},
			ConfigKey:   "sys.rateLimit.login.windowSecs",
			ConfigValue: "60",
			ConfigType:  "N",
			Remark:      ptr.To("登录接口限流窗口大小(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
	}

	for _, cfg := range configs {
		if err := tx.Where(entity.SysConfig{ConfigKey: cfg.ConfigKey}).FirstOrCreate(&cfg).Error; err != nil {
			return err
		}
	}

	return nil
}
