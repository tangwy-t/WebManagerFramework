package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
)

func init() {
	migration.Register(migration.Migration{
		Version:     3,
		Description: "授予 admin 角色全部菜单权限",
		Up:          seedPermissionsV3,
	})
}

// seedPermissionsV3 grants all menu permissions to the admin role (role_id=1).
// It batch-queries all menu IDs and existing associations, then only inserts
// the missing ones using CreateInBatches for efficiency.
func seedPermissionsV3(tx *gorm.DB) error {
	// Batch get all menu IDs
	var menuIDs []uint64
	if err := tx.Model(&entity.SysMenu{}).Pluck("id", &menuIDs).Error; err != nil {
		return err
	}

	if len(menuIDs) == 0 {
		return nil
	}

	// Batch get existing associations for admin role (role_id=1)
	var existing []entity.SysRoleMenu
	if err := tx.Where("role_id = ?", 1).Find(&existing).Error; err != nil {
		return err
	}

	existingSet := make(map[uint64]bool, len(existing))
	for _, assoc := range existing {
		existingSet[assoc.MenuID] = true
	}

	// Only insert missing associations
	var toCreate []entity.SysRoleMenu
	for _, mid := range menuIDs {
		if !existingSet[mid] {
			toCreate = append(toCreate, entity.SysRoleMenu{
				RoleID: 1,
				MenuID: mid,
			})
		}
	}

	if len(toCreate) == 0 {
		return nil
	}

	return tx.CreateInBatches(toCreate, 100).Error
}
