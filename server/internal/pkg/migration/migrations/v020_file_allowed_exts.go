package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
)

// v006 种子原值:仅覆盖图片/文档/压缩包中的少量扩展名。
// 文件管理模块按 7 个分类(与 entity.FileCategoryExts 对齐)组织 UI,
// 原白名单下代码、音视频及大部分文档/压缩包扩展名均无法上传,
// 分类筛选与分布条在演示环境失去意义。本迁移在配置仍为种子原值时
// 扩展为完整清单;管理员后续可在"参数配置"页面自行收紧(仅覆盖原值,
// 用户自定义值保持不变)。
const (
	fileAllowedExtsSeedV6 = ".jpg,.jpeg,.png,.gif,.webp,.pdf,.doc,.docx,.xls,.xlsx,.txt,.csv,.zip"
	fileAllowedExtsFull   = ".jpg,.jpeg,.png,.gif,.webp,.svg,.bmp,.ico,.avif,.mp4,.avi,.mov,.mkv,.webm,.flv,.wmv,.m4v,.rmvb,.mp3,.wav,.flac,.aac,.ogg,.wma,.m4a,.amr,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.md,.csv,.rtf,.odt,.zip,.rar,.7z,.tar,.gz,.bz2,.xz,.js,.ts,.jsx,.tsx,.vue,.py,.go,.java,.c,.cpp,.h,.html,.css,.scss,.json,.xml,.yml,.yaml,.sql,.sh,.toml"
)

func init() {
	migration.Register(migration.Migration{
		Version:     20,
		Description: "扩展文件上传白名单:与文件管理 7 分类扩展名清单对齐",
		Up:          expandFileAllowedExtsV20,
	})
}

func expandFileAllowedExtsV20(tx *gorm.DB) error {
	return tx.Model(&entity.SysConfig{}).
		Where("config_key = ? AND config_value = ?", "sys.file.upload.allowedExts", fileAllowedExtsSeedV6).
		Update("config_value", fileAllowedExtsFull).Error
}