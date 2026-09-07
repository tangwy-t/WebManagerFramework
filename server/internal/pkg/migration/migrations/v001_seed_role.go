package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     1,
		Description: "初始化角色",
		Up:          seedRole,
	})
}

// adminRole 超级管理员角色：后续菜单授权、用户绑定均引用此变量的 snowflake ID。
var adminRole = entity.SysRole{
	Name:      "超级管理员",
	Code:      "admin",
	DataScope: ptr.To[int8](1),
	Sort:      ptr.To(0),
	Status:    ptr.To[int8](entity.RoleStatusEnabled),
	Remark:    ptr.To("系统内置超级管理员角色"),
}

func seedRole(tx *gorm.DB) error {
	role := adminRole
	if err := tx.CreateInBatches([]entity.SysRole{role}, 1).Error; err != nil {
		return err
	}
	adminRoleID = role.ID // snowflake 回调已回写
	return nil
}
