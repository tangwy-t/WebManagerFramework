package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
)

func init() {
	migration.Register(migration.Migration{
		Version:     3,
		Description: "初始化超级管理员菜单权限",
		Up:          seedPermission,
	})
}

// 由前序迁移填充的共享状态。
var (
	adminRoleID uint64   // seedRole 创建后写入
	allMenuIDs  []uint64 // seedMenu 创建后写入
)

// seedPermission 把全部菜单授权给超级管理员角色，批量插入 sys_role_menu。
func seedPermission(tx *gorm.DB) error {
	if len(allMenuIDs) == 0 {
		return nil
	}
	roleMenus := make([]entity.SysRoleMenu, 0, len(allMenuIDs))
	for _, mid := range allMenuIDs {
		roleMenus = append(roleMenus, entity.SysRoleMenu{
			RoleID: adminRoleID,
			MenuID: mid,
		})
	}
	return tx.CreateInBatches(roleMenus, 100).Error
}
