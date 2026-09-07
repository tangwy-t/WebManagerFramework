package migrations

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/crypto"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     4,
		Description: "创建 admin 用户并绑定超级管理员角色",
		Up:          seedAdminUser,
	})
}

// defaultAdminPassword 是 admin 用户的初始密码（写死，便于首次登录）。
// 上线后务必立即登录并在“修改密码”中更换为强密码。
const defaultAdminPassword = "admin123"

// seedAdminUser 创建 admin 用户（密码哈希需 bcrypt，唯一非纯 INSERT 步骤），
// 并将其绑定到超级管理员角色。
func seedAdminUser(tx *gorm.DB) error {
	hashed, salt, err := crypto.HashPassword(defaultAdminPassword, bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	status := entity.UserStatusEnabled
	user := entity.SysUser{
		Username:     "admin",
		Password:     hashed,
		PasswordSalt: ptr.To(salt),
		Status:       &status,
	}
	if err := tx.Create(&user).Error; err != nil {
		return err
	}

	// 种子间不共享包级变量：按 code 自行查询 v001 创建的超管角色 ID。
	var adminRoleID uint64
	if err := tx.Model(&entity.SysRole{}).
		Select("id").
		Where("code = ?", "admin").
		First(&adminRoleID).Error; err != nil {
		return fmt.Errorf("seedAdminUser: 查询超级管理员角色失败: %w", err)
	}

	if err := tx.Create(&entity.SysUserRole{
		UserID: user.ID,
		RoleID: adminRoleID,
	}).Error; err != nil {
		return err
	}
	return nil
}
