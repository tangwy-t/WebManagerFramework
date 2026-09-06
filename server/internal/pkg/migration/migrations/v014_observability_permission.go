package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     14,
		Description: "新增pprof菜单和 system:user:unlock 权限码",
		Up:          seedObservabilityPermissionV14,
	})
}

// seedObservabilityPermissionV14 插入以下菜单和权限项，并授权给 admin 角色：
//   - 解锁用户 btn (ID 81, parent=2 用户管理)
//   - pprof menu (ID 82, parent=74 服务监控)
//   - 启用pprof btn (ID 83, parent=82)
//   - 停用pprof btn (ID 84, parent=82)
//   - 同时补上缺失的停用用户 (ID 80) 的 sys_role_menu 授权
//   - 播种 sys.pprof.enabled / sys.pprof.autoOffSeconds 配置项
func seedObservabilityPermissionV14(tx *gorm.DB) error {
	// 1. 解锁用户按钮 (ID 81, parent=2 用户管理)
	btnUnlock := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 81},
		ParentID:   ptr.To(uint64(2)),
		Name:       "解锁用户",
		Type:       "btn",
		Perms:      ptr.To("system:user:unlock"),
		Sort:       ptr.To(8),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 81}}).FirstOrCreate(&btnUnlock).Error; err != nil {
		return err
	}

	// 2. pprof 菜单项 (ID 82, parent=74 服务监控)
	menuPprof := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 82},
		ParentID:   ptr.To(uint64(74)),
		Name:       "pprof",
		Type:       "menu",
		Path:       ptr.To("/monitor/pprof"),
		Component:  ptr.To("monitor/pprof/index"),
		Perms:      ptr.To("system:pprof:list"),
		Sort:       ptr.To(2),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 82}}).FirstOrCreate(&menuPprof).Error; err != nil {
		return err
	}

	// 3. 启用pprof 按钮 (ID 83, parent=82)
	btnEnable := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 83},
		ParentID:   ptr.To(uint64(82)),
		Name:       "启用pprof",
		Type:       "btn",
		Perms:      ptr.To("system:pprof:enable"),
		Sort:       ptr.To(1),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 83}}).FirstOrCreate(&btnEnable).Error; err != nil {
		return err
	}

	// 4. 停用pprof 按钮 (ID 84, parent=82)
	btnDisable := entity.SysMenu{
		BaseEntity: entity.BaseEntity{ID: 84},
		ParentID:   ptr.To(uint64(82)),
		Name:       "停用pprof",
		Type:       "btn",
		Perms:      ptr.To("system:pprof:disable"),
		Sort:       ptr.To(2),
		Visible:    ptr.To[int8](1),
		Status:     ptr.To[int8](1),
	}
	if err := tx.Where(entity.SysMenu{BaseEntity: entity.BaseEntity{ID: 84}}).FirstOrCreate(&btnDisable).Error; err != nil {
		return err
	}

	// 5. 授权给 admin 角色 (role_id=1)
	// 同时补上停用用户 (ID 80) 的缺失授权
	roleMenus := []entity.SysRoleMenu{
		{ID: 71, RoleID: 1, MenuID: 80}, // 停用用户（补缺失授权）
		{ID: 72, RoleID: 1, MenuID: 81}, // 解锁用户
		{ID: 73, RoleID: 1, MenuID: 82}, // pprof
		{ID: 74, RoleID: 1, MenuID: 83}, // 启用pprof
		{ID: 75, RoleID: 1, MenuID: 84}, // 停用pprof
	}
	for _, rm := range roleMenus {
		if err := tx.Where(entity.SysRoleMenu{ID: rm.ID}).FirstOrCreate(&rm).Error; err != nil {
			return err
		}
	}

	// 6. 播种 observability 配置项
	configs := []entity.SysConfig{
		{
			BaseEntity:  entity.BaseEntity{ID: 31},
			ConfigKey:   "sys.pprof.enabled",
			ConfigValue: "false",
			ConfigType:  "B",
			Remark:      ptr.To("pprof 是否启用（运行时动态切换）"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
		{
			BaseEntity:  entity.BaseEntity{ID: 32},
			ConfigKey:   "sys.pprof.autoOffSeconds",
			ConfigValue: "300",
			ConfigType:  "N",
			Remark:      ptr.To("pprof 自动关闭超时（秒），0 表示不自动关闭"),
			Status:      ptr.To[int8](entity.ConfigStatusEnabled),
		},
	}
	for _, cfg := range configs {
		if err := tx.Where(entity.SysConfig{ConfigKey: cfg.ConfigKey}).FirstOrCreate(&cfg).Error; err != nil {
			return err
		}
	}

	return nil
}
