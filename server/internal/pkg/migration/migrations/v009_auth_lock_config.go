package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     9,
		Description: "写入账号锁定配置项，标记 captchaFailWindowSeconds 为废弃",
		Up:          seedAuthLockConfigV9,
	})
}

func seedAuthLockConfigV9(tx *gorm.DB) error {
	configs := []entity.SysConfig{
		{
			BaseEntity:  entity.BaseEntity{ID: 22},
			ConfigKey:   "sys.auth.lockThreshold",
			ConfigValue: "10",
			ConfigType:  "N",
			Remark:      ptr.To("触发账号锁定的连续登录失败次数"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 23},
			ConfigKey:   "sys.auth.lockDuration",
			ConfigValue: "30",
			ConfigType:  "N",
			Remark:      ptr.To("账号锁定持续时间(分钟)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 24},
			ConfigKey:   "sys.auth.lockWindow",
			ConfigValue: "15",
			ConfigType:  "N",
			Remark:      ptr.To("登录失败计数窗口(分钟)，替代 captchaFailWindowSeconds"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
	}

	for _, cfg := range configs {
		if err := tx.Where(entity.SysConfig{ConfigKey: cfg.ConfigKey}).FirstOrCreate(&cfg).Error; err != nil {
			return err
		}
	}

	// Update captchaFailWindowSeconds remark to deprecated
	deprecatedRemark := "已废弃: 被 sys.auth.lockWindow 替代 (单位已改为分钟)"
	if err := tx.Model(&entity.SysConfig{}).
		Where("config_key = ?", "sys.auth.captchaFailWindowSeconds").
		Update("remark", &deprecatedRemark).Error; err != nil {
		return err
	}

	return nil
}
