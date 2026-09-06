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
		Description: "创建 admin 角色",
		Up:          seedAdminRole,
	})
}

func seedAdminRole(tx *gorm.DB) error {
	role := entity.SysRole{
		BaseEntity: entity.BaseEntity{ID: 1},
		Name:       "超级管理员",
		Code:       "admin",
		DataScope:  ptr.To[int8](1),
		Sort:       ptr.To(0),
		Status:     ptr.To[int8](1),
		Remark:     ptr.To("系统内置超级管理员角色"),
	}
	return tx.Where(entity.SysRole{Code: "admin"}).FirstOrCreate(&role).Error
}
