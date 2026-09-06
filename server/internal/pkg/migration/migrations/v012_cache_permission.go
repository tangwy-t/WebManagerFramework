package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     12,
		Description: "新增缓存管理菜单和 system:cache:* 权限码",
		Up:          seedCachePermissionV12,
	})
}

// seedCachePermissionV12 插入缓存管理菜单项和按钮权限，并授权给 admin 角色。
// 菜单项 (ID 76) 挂在服务监控 (ID 74) 下。
func seedCachePermissionV12(tx *gorm.DB) error {
	// 1. 插入缓存管理菜单项 (ID 76, parent=74 服务监控)
	menu := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 76},
		ParentID:   ptr.To(uint64(74)),
		Name:       "缓存管理",
		Type:       "menu",
		Path:       ptr.To("/monitor/cache"),
		Perms:      ptr.To("system:cache:list"),
		Sort:       ptr.To(4),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 76}}).FirstOrCreate(&menu).Error; err != nil {
		return err
	}

	// 2. 插入缓存查询按钮 (ID 77, parent=76)
	btnQuery := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 77},
		ParentID:   ptr.To(uint64(76)),
		Name:       "缓存查询",
		Type:       "btn",
		Perms:      ptr.To("system:cache:query"),
		Sort:       ptr.To(1),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 77}}).FirstOrCreate(&btnQuery).Error; err != nil {
		return err
	}

	// 3. 插入缓存删除按钮 (ID 78, parent=76)
	btnDelete := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 78},
		ParentID:   ptr.To(uint64(76)),
		Name:       "缓存删除",
		Type:       "btn",
		Perms:      ptr.To("system:cache:delete"),
		Sort:       ptr.To(2),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 78}}).FirstOrCreate(&btnDelete).Error; err != nil {
		return err
	}

	// 4. 授权给 admin 角色 (role_id=1)
	roleMenus := []entity.SysRoleMenu{
		{ID: 67, RoleID: 1, MenuID: 76},
		{ID: 68, RoleID: 1, MenuID: 77},
		{ID: 69, RoleID: 1, MenuID: 78},
	}
	for _, rm := range roleMenus {
		if err := tx.Where(entity.SysRoleMenu{ID: rm.ID}).FirstOrCreate(&rm).Error; err != nil {
			return err
		}
	}

	return nil
}
