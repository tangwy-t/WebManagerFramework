package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     6,
		Description: "写入默认系统配置项",
		Up:          seedConfigV6,
	})
}

func seedConfigV6(tx *gorm.DB) error {
	configs := []entity.SysConfig{
		{
			BaseEntity: entity.BaseEntity{ID: 1},
			ConfigKey:  "sys.jwt.secret",
			// 安全说明：这是一个公开的占位值（满足 32 字节校验）。wireup.Init
			// 启动时会调用 ConfigService.RotateDefaultJWTSecret 检出该值并立即
			// 轮换为随机密钥（含 Redis 缓存刷新与跨实例通知）。不要移除该
			// 检测逻辑；生产密钥请直接经配置接口写入。
			ConfigValue: "default-jwt-secret-key-at-least-32-bytes",
			ConfigType:  "S",
			Remark:      ptr.To("JWT签名密钥"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 2},
			ConfigKey:   "sys.jwt.accessExpire",
			ConfigValue: "7200",
			ConfigType:  "N",
			Remark:      ptr.To("访问令牌过期时间(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 3},
			ConfigKey:   "sys.jwt.refreshExpire",
			ConfigValue: "604800",
			ConfigType:  "N",
			Remark:      ptr.To("刷新令牌过期时间(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 4},
			ConfigKey:   "sys.jwt.issuer",
			ConfigValue: "server",
			ConfigType:  "S",
			Remark:      ptr.To("JWT签发者"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 5},
			ConfigKey:   "sys.file.upload.maxSize",
			ConfigValue: "10485760",
			ConfigType:  "N",
			Remark:      ptr.To("上传文件最大大小(字节)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 6},
			ConfigKey:   "sys.file.upload.path",
			ConfigValue: "./uploads",
			ConfigType:  "S",
			Remark:      ptr.To("上传文件存储路径"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 7},
			ConfigKey:   "sys.file.upload.allowedExts",
			ConfigValue: ".jpg,.jpeg,.png,.gif,.webp,.pdf,.doc,.docx,.xls,.xlsx,.txt,.csv,.zip",
			ConfigType:  "S",
			Remark:      ptr.To("允许上传的文件扩展名"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 8},
			ConfigKey:   "sys.scheduler.enabled",
			ConfigValue: "true",
			ConfigType:  "B",
			Remark:      ptr.To("调度器是否启用"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 9},
			ConfigKey:   "sys.scheduler.stopTimeout",
			ConfigValue: "30",
			ConfigType:  "N",
			Remark:      ptr.To("调度器停止超时(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 10},
			ConfigKey:   "sys.scheduler.lockTTL",
			ConfigValue: "60",
			ConfigType:  "N",
			Remark:      ptr.To("任务锁TTL(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 11},
			ConfigKey:   "sys.scheduler.maxExecutionTime",
			ConfigValue: "300",
			ConfigType:  "N",
			Remark:      ptr.To("任务最大执行时间(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 12},
			ConfigKey:   "sys.scheduler.maxRetryCount",
			ConfigValue: "3",
			ConfigType:  "N",
			Remark:      ptr.To("任务最大重试次数"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 13},
			ConfigKey:   "sys.scheduler.resyncInterval",
			ConfigValue: "60",
			ConfigType:  "N",
			Remark:      ptr.To("调度器同步间隔(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 14},
			ConfigKey:   "sys.auth.asyncLogTimeout",
			ConfigValue: "5",
			ConfigType:  "N",
			Remark:      ptr.To("异步日志超时(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 15},
			ConfigKey:   "sys.auth.bcryptCost",
			ConfigValue: "10",
			ConfigType:  "N",
			Remark:      ptr.To("密码哈希成本"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 16},
			ConfigKey:   "sys.log.retentionDays",
			ConfigValue: "90",
			ConfigType:  "N",
			Remark:      ptr.To("日志保留天数"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 17},
			ConfigKey:   "sys.config.resyncInterval",
			ConfigValue: "86400",
			ConfigType:  "N",
			Remark:      ptr.To("配置参数同步间隔(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 18},
			ConfigKey:   "sys.auth.captchaFailThreshold",
			ConfigValue: "5",
			ConfigType:  "N",
			Remark:      ptr.To("触发验证码的登录失败次数阈值"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 19},
			ConfigKey:   "sys.auth.captchaExpireSeconds",
			ConfigValue: "300",
			ConfigType:  "N",
			Remark:      ptr.To("验证码有效期(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 20},
			ConfigKey:   "sys.auth.captchaFailWindowSeconds",
			ConfigValue: "900",
			ConfigType:  "N",
			Remark:      ptr.To("登录失败计数窗口(秒)"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 21},
			ConfigKey:   "sys.auth.captchaRateLimitPerMinute",
			ConfigValue: "10",
			ConfigType:  "N",
			Remark:      ptr.To("同IP每分钟最大验证码获取次数"),
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
