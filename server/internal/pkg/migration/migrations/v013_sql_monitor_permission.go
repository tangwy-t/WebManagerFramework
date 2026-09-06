package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     13,
		Description: "新增SQL监控菜单和 system:sql:list 权限码",
		Up:          seedSQLMonitorPermissionV13,
	})
}

// seedSQLMonitorPermissionV13 插入 SQL 监控菜单项并授权给 admin 角色。
// 菜单项 (ID 79) 挂在服务监控 (ID 74) 下。
func seedSQLMonitorPermissionV13(tx *gorm.DB) error {
	// 1. 插入 SQL 监控菜单项 (ID 79, parent=74 服务监控)
	menu := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 79},
		ParentID:   ptr.To(uint64(74)),
		Name:       "SQL监控",
		Type:       "menu",
		Path:       ptr.To("/monitor/sql"),
		Perms:      ptr.To("system:sql:list"),
		Sort:       ptr.To(5),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 79}}).FirstOrCreate(&menu).Error; err != nil {
		return err
	}

	// 2. 授权给 admin 角色 (role_id=1)
	roleMenu := entity.SysRoleMenu{
		ID:     70,
		RoleID: 1,
		MenuID: 79,
	}
	if err := tx.Where(entity.SysRoleMenu{ID: 70}).FirstOrCreate(&roleMenu).Error; err != nil {
		return err
	}

	return nil
}
