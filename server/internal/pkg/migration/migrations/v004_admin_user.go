package migrations

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/crypto"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

// defaultAdminPassword 是 admin 用户的初始密码（写死，便于首次登录）。
// 上线后务必立即登录并在"修改密码"中更换为强密码。
// 仅在 admin 尚未创建时由 v004 生效；已存在的账户请走修改密码流程重置。
const defaultAdminPassword = "admin123"

func init() {
	migration.Register(migration.Migration{
		Version:     4,
		Description: "创建 admin 用户",
		Up:          seedAdminUserV4,
	})
}

func seedAdminUserV4(tx *gorm.DB) error {
	// 检查 admin 用户是否已存在
	var existing entity.SysUser
	if err := tx.Where("username = ?", "admin").First(&existing).Error; err == nil {
		return nil // 已存在，跳过
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("migration v004: 查询 admin 用户失败: %w", err)
	}

	// 初始密码写死为源码常量，无需输出到日志/stdout：
	// 旧实现用 fmt.Printf 打印随机密码，但只打印一次且走 stdout，
	// 后台运行时极易丢失（server.log 里也查不到）。
	// 登录后请立即通过"修改密码"功能更换。
	password := defaultAdminPassword
	hashed, salt, err := crypto.HashPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("migration v004: 密码哈希失败: %w", err)
	}

	status := entity.UserStatusEnabled
	adminUser := entity.SysUser{
		Username:     "admin",
		Password:     hashed,
		PasswordSalt: ptr.To(salt),
		Status:       &status,
	}
	if err := tx.Create(&adminUser).Error; err != nil {
		return fmt.Errorf("migration v004: 创建 admin 用户失败: %w", err)
	}

	// 分配 admin 角色
	userRole := entity.SysUserRole{
		UserID: adminUser.ID,
		RoleID: 1,
	}
	if err := tx.Where("user_id = ? AND role_id = ?", adminUser.ID, 1).FirstOrCreate(&userRole).Error; err != nil {
		return fmt.Errorf("migration v004: 分配角色失败: %w", err)
	}

	return nil
}
