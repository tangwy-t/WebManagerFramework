package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     11,
		Description: "新增服务监控目录和 system:server:list 权限码",
		Up:          seedMonitorPermissionV11,
	})
}

// seedMonitorPermissionV11 插入服务监控目录和服务器监控菜单项，并授权给 admin 角色。
// 服务监控目录 (ID 74) 挂在系统管理 (ID 1) 下。
// 服务器监控菜单 (ID 75) 挂在服务监控 (ID 74) 下。
func seedMonitorPermissionV11(tx *gorm.DB) error {
	// 1. 插入服务监控目录 (ID 74, parent=1 系统管理)
	dir := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 74},
		ParentID:   ptr.To(uint64(1)),
		Name:       "服务监控",
		Type:       "dir",
		Sort:       ptr.To(10),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 74}}).FirstOrCreate(&dir).Error; err != nil {
		return err
	}

	// 2. 插入服务器监控菜单项 (ID 75, parent=74 服务监控)
	menu := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 75},
		ParentID:   ptr.To(uint64(74)),
		Name:       "服务器监控",
		Type:       "menu",
		Path:       ptr.To("/monitor/server"),
		Component:  ptr.To("monitor/server/index"),
		Perms:      ptr.To("system:server:list"),
		Sort:       ptr.To(1),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 75}}).FirstOrCreate(&menu).Error; err != nil {
		return err
	}

	// 3. 授权给 admin 角色 (role_id=1)
	roleMenus := []entity.SysRoleMenu{
		{ID: 74, RoleID: 1, MenuID: 74},
		{ID: 75, RoleID: 1, MenuID: 75},
	}
	for _, rm := range roleMenus {
		if err := tx.Where(entity.SysRoleMenu{ID: rm.ID}).FirstOrCreate(&rm).Error; err != nil {
			return err
		}
	}

	return nil
}
