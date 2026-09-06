package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
)

func init() {
	migration.Register(migration.Migration{
		Version:     17,
		Description: "登录日志结果码:status→code 回填 + 字典 10001 文案改为“认证失败”(兼容登录失败与接口 401 两类场景)",
		Up:          migrateLoginLogCodeV17,
	})
}

func migrateLoginLogCodeV17(tx *gorm.DB) error {
	// 1. 字典 10001 标签改为“认证失败”:操作日志里它代表 401 未登录/令牌过期,
	if err := tx.Table("sys_dict_data").
		Where("type_id = (SELECT id FROM sys_dict_type WHERE code = 'sys_opt_result_code' LIMIT 1) AND value = '10001'").
		Update("label", "认证失败").Error; err != nil {
		return err
	}
	return nil
}
