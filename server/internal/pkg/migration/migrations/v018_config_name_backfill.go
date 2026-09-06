package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
)

func init() {
	migration.Register(migration.Migration{
		Version:     18,
		Description: "为既有系统配置项回填名称(name 字段)",
		Up:          backfillConfigNamesV18,
	})
}

// backfillConfigNamesV18 为内建系统配置项回填 name(该字段此前不存在)。
// 仅更新 name 为空的记录,用户自建配置保持原样、不臆造名称。
// 同时覆盖全新安装(种子迁移之后运行)与升级安装(历史数据)两种情况。
func backfillConfigNamesV18(tx *gorm.DB) error {
	names := map[string]string{
		"sys.jwt.secret":                     "JWT签名密钥",
		"sys.jwt.accessExpire":               "访问令牌过期时间",
		"sys.jwt.refreshExpire":              "刷新令牌过期时间",
		"sys.jwt.issuer":                     "JWT签发者",
		"sys.file.upload.maxSize":            "上传文件最大大小",
		"sys.file.upload.path":               "上传文件存储路径",
		"sys.file.upload.allowedExts":        "允许上传文件扩展名",
		"sys.scheduler.enabled":              "调度器开关",
		"sys.scheduler.stopTimeout":          "调度器停止超时",
		"sys.scheduler.lockTTL":              "任务锁TTL",
		"sys.scheduler.maxExecutionTime":     "任务最大执行时间",
		"sys.scheduler.maxRetryCount":        "任务最大重试次数",
		"sys.scheduler.resyncInterval":       "调度器同步间隔",
		"sys.auth.asyncLogTimeout":           "异步日志超时",
		"sys.auth.bcryptCost":                "密码哈希成本",
		"sys.log.retentionDays":              "日志保留天数",
		"sys.config.resyncInterval":          "配置同步间隔",
		"sys.auth.captchaFailThreshold":      "验证码触发阈值",
		"sys.auth.captchaExpireSeconds":      "验证码有效期",
		"sys.auth.captchaFailWindowSeconds":  "失败计数窗口",
		"sys.auth.captchaRateLimitPerMinute": "验证码获取限流",
		"sys.auth.lockThreshold":             "锁定失败次数阈值",
		"sys.auth.lockDuration":              "锁定持续时间",
		"sys.auth.lockWindow":                "失败计数窗口",
		"sys.rateLimit.global.enabled":       "全局限流开关",
		"sys.rateLimit.global.limit":         "全局每窗口最大请求数",
		"sys.rateLimit.global.windowSecs":    "全局限流窗口大小",
		"sys.rateLimit.login.enabled":        "登录限流开关",
		"sys.rateLimit.login.limit":          "登录每窗口最大请求数",
		"sys.rateLimit.login.windowSecs":     "登录限流窗口大小",
	}
	for key, name := range names {
		if err := tx.Model(&entity.SysConfig{}).
			Where("config_key = ? AND (name IS NULL OR name = '')", key).
			Update("name", name).Error; err != nil {
			return err
		}
	}
	return nil
}