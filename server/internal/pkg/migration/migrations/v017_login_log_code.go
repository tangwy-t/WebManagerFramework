package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
)

func init() {
	migration.Register(migration.Migration{
		Version:     17,
		Description: "登录日志结果码:status→code 回填 + 字典 10001 文案改为“认证失败”(兼容登录失败与接口 401 两类场景)",
		Up:          migrateLoginLogCodeV17,
	})
}

// loginLogMsgCodeMap 历史登录失败文案 → 业务码映射(登录埋点历史文案固定)。
var loginLogMsgCodeMap = map[string]int{
	"用户名或密码错误": apperror.CodeUnauthorized,
	"账号已被禁用":   apperror.CodeUnauthorized,
	"账号已锁定":    apperror.CodeAccountLocked,
	"验证码错误":    apperror.CodeCaptchaIncorrect,
}

func migrateLoginLogCodeV17(tx *gorm.DB) error {
	// 1. 字典 10001 标签改为“认证失败”:操作日志里它代表 401 未登录/令牌过期,
	//    登录日志里代表“用户名或密码错误”,原“未登录/令牌缺失”在登录场景下不通。
	if err := tx.Table("sys_dict_data").
		Where("type_id = (SELECT id FROM sys_dict_type WHERE code = 'sys_opt_result_code' LIMIT 1) AND value = '10001'").
		Update("label", "认证失败").Error; err != nil {
		return err
	}

	// 2. 登录日志失败行回填:旧 status=0(失败)且 code 仍为默认 0 的行,
	//    按 msg 文案映射到具体业务码;未知文案兜底 10001。
	//    成功行(status=1)code 默认 0 即正确,无需处理。
	var legacy []struct {
		ID  uint64
		Msg *string
	}
	if err := tx.Table("sys_login_log").
		Where("code = 0 AND status = 0").
		Select("id", "msg").Find(&legacy).Error; err != nil {
		return err
	}
	for _, row := range legacy {
		code := apperror.CodeUnauthorized
		if row.Msg != nil {
			if c, ok := loginLogMsgCodeMap[*row.Msg]; ok {
				code = c
			}
		}
		if err := tx.Table("sys_login_log").Where("id = ?", row.ID).Update("code", code).Error; err != nil {
			return err
		}
	}

	return nil
}