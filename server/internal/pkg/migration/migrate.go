package migration

import (
	"github.com/tangwy-t/webmanager-server/internal/model/entity"

	"gorm.io/gorm"
)

// MigrateAll 执行全部实体的 AutoMigrate。
// 新增实体时只需在此处追加，无需修改 main()。
func MigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.SysUser{},
		&entity.SysRole{},
		&entity.SysDept{},
		&entity.SysMenu{},
		&entity.SysUserRole{},
		&entity.SysRoleMenu{},
		&entity.SysRoleDept{},
		&entity.SysDictType{},
		&entity.SysDictData{},
		&entity.SysConfig{},
		&entity.SysNotice{},
		&entity.SysNoticeUser{},
		&entity.SysFile{},
		&entity.SysJob{},
		&entity.SysJobLog{},
		&entity.SysOperationLog{},
		&entity.SysLoginLog{},
		&entity.SysMigration{},
	)
}
