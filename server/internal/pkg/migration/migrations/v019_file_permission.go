package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     19,
		Description: "补齐文件管理按钮权限(下载 system:file:download / 编辑 system:file:edit)并授权 admin",
		Up:          seedFileButtonPermsV19,
	})
}

// seedFileButtonPermsV19 在文件管理菜单(ID 48,v002 已种子)下补齐按钮权限:
// v002 仅种了 list/upload/delete/query,文件管理 UI 的下载与重命名动作
// 需要独立权限码,故追加两个 btn 节点并授权给 admin 角色(role_id=1)。
func seedFileButtonPermsV19(tx *gorm.DB) error {
	// 1. 下载/编辑按钮节点(挂在文件管理菜单 48 下)
	menus := []entity.SysMenu{
		{
			BaseEntity: entity.BaseEntity{ID: 347},
			ParentID:   ptr.To(uint64(48)),
			Name:       "下载文件",
			Type:       "btn",
			Perms:      ptr.To("system:file:download"),
			Sort:       ptr.To(3),
			Visible:    ptr.To[int8](1),
			Status:     ptr.To[int8](1),
		},
		{
			BaseEntity: entity.BaseEntity{ID: 348},
			ParentID:   ptr.To(uint64(48)),
			Name:       "编辑文件",
			Type:       "btn",
			Perms:      ptr.To("system:file:edit"),
			Sort:       ptr.To(4),
			Visible:    ptr.To[int8](1),
			Status:     ptr.To[int8](1),
		},
	}
	for i := range menus {
		m := menus[i]
		if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: m.ID}}).FirstOrCreate(&m).Error; err != nil {
			return err
		}
	}

	// 2. 授权给 admin 角色(role_id=1)。
	roleMenus := []entity.SysRoleMenu{
		{RoleID: 1, MenuID: 347},
		{RoleID: 1, MenuID: 348},
	}
	for _, rm := range roleMenus {
		if err := tx.Where(entity.SysRoleMenu{RoleID: rm.RoleID, MenuID: rm.MenuID}).FirstOrCreate(&rm).Error; err != nil {
			return err
		}
	}

	return nil
}