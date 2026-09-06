package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     2,
		Description: "种子系统菜单",
		Up:          seedMenusV2,
	})
}

// menusV2 contains all system menu items extracted from the old seeder.
var menusV2 = []entity.SysMenu{
	// ── 系统管理 (dir) ─────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 1}, ParentID: ptr.To(uint64(0)), Name: "系统管理", Type: "dir", Sort: ptr.To(0), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 用户管理 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 2}, ParentID: ptr.To(uint64(1)), Name: "用户管理", Type: "menu", Path: ptr.To("/system/user"), Component: ptr.To("system/user/index"), Perms: ptr.To("system:user:list"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 3}, ParentID: ptr.To(uint64(2)), Name: "用户查询", Type: "btn", Perms: ptr.To("system:user:query"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 4}, ParentID: ptr.To(uint64(2)), Name: "用户新增", Type: "btn", Perms: ptr.To("system:user:add"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 5}, ParentID: ptr.To(uint64(2)), Name: "用户修改", Type: "btn", Perms: ptr.To("system:user:edit"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 6}, ParentID: ptr.To(uint64(2)), Name: "用户删除", Type: "btn", Perms: ptr.To("system:user:delete"), Sort: ptr.To(4), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 72}, ParentID: ptr.To(uint64(2)), Name: "启用用户", Type: "btn", Perms: ptr.To("system:user:enable"), Sort: ptr.To(5), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 80}, ParentID: ptr.To(uint64(2)), Name: "停用用户", Type: "btn", Perms: ptr.To("system:user:disable"), Sort: ptr.To(6), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 73}, ParentID: ptr.To(uint64(2)), Name: "重置密码", Type: "btn", Perms: ptr.To("system:user:reset"), Sort: ptr.To(7), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 角色管理 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 7}, ParentID: ptr.To(uint64(1)), Name: "角色管理", Type: "menu", Path: ptr.To("/system/role"), Component: ptr.To("system/role/index"), Perms: ptr.To("system:role:list"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 8}, ParentID: ptr.To(uint64(7)), Name: "角色查询", Type: "btn", Perms: ptr.To("system:role:query"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 9}, ParentID: ptr.To(uint64(7)), Name: "角色新增", Type: "btn", Perms: ptr.To("system:role:add"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 10}, ParentID: ptr.To(uint64(7)), Name: "角色修改", Type: "btn", Perms: ptr.To("system:role:edit"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 11}, ParentID: ptr.To(uint64(7)), Name: "角色删除", Type: "btn", Perms: ptr.To("system:role:delete"), Sort: ptr.To(4), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 菜单管理 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 12}, ParentID: ptr.To(uint64(1)), Name: "菜单管理", Type: "menu", Path: ptr.To("/system/menu"), Component: ptr.To("system/menu/index"), Perms: ptr.To("system:menu:list"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 13}, ParentID: ptr.To(uint64(12)), Name: "菜单查询", Type: "btn", Perms: ptr.To("system:menu:query"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 14}, ParentID: ptr.To(uint64(12)), Name: "菜单新增", Type: "btn", Perms: ptr.To("system:menu:add"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 15}, ParentID: ptr.To(uint64(12)), Name: "菜单修改", Type: "btn", Perms: ptr.To("system:menu:edit"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 16}, ParentID: ptr.To(uint64(12)), Name: "菜单删除", Type: "btn", Perms: ptr.To("system:menu:delete"), Sort: ptr.To(4), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 部门管理 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 17}, ParentID: ptr.To(uint64(1)), Name: "部门管理", Type: "menu", Path: ptr.To("/system/dept"), Component: ptr.To("system/dept/index"), Perms: ptr.To("system:dept:list"), Sort: ptr.To(4), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 18}, ParentID: ptr.To(uint64(17)), Name: "部门查询", Type: "btn", Perms: ptr.To("system:dept:query"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 19}, ParentID: ptr.To(uint64(17)), Name: "部门新增", Type: "btn", Perms: ptr.To("system:dept:add"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 20}, ParentID: ptr.To(uint64(17)), Name: "部门修改", Type: "btn", Perms: ptr.To("system:dept:edit"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 21}, ParentID: ptr.To(uint64(17)), Name: "部门删除", Type: "btn", Perms: ptr.To("system:dept:delete"), Sort: ptr.To(4), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 日志管理 (dir) ─────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 22}, ParentID: ptr.To(uint64(1)), Name: "日志管理", Type: "dir", Sort: ptr.To(5), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 操作日志 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 23}, ParentID: ptr.To(uint64(22)), Name: "操作日志", Type: "menu", Path: ptr.To("/monitor/operlog"), Component: ptr.To("monitor/operlog/index"), Perms: ptr.To("system:log:operation:list"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 24}, ParentID: ptr.To(uint64(23)), Name: "操作日志查询", Type: "btn", Perms: ptr.To("system:log:operation:list"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 25}, ParentID: ptr.To(uint64(23)), Name: "操作日志删除", Type: "btn", Perms: ptr.To("system:log:operation:delete"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 登录日志 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 26}, ParentID: ptr.To(uint64(22)), Name: "登录日志", Type: "menu", Path: ptr.To("/monitor/logininfor"), Component: ptr.To("monitor/logininfor/index"), Perms: ptr.To("system:log:login:list"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 27}, ParentID: ptr.To(uint64(26)), Name: "登录日志查询", Type: "btn", Perms: ptr.To("system:log:login:list"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 字典管理 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 51}, ParentID: ptr.To(uint64(1)), Name: "字典管理", Type: "menu", Path: ptr.To("/system/dict"), Component: ptr.To("system/dict/index"), Perms: ptr.To("system:dict:list"), Sort: ptr.To(6), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 52}, ParentID: ptr.To(uint64(51)), Name: "查询字典类型", Type: "btn", Perms: ptr.To("system:dict:type:query"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 53}, ParentID: ptr.To(uint64(51)), Name: "新增字典类型", Type: "btn", Perms: ptr.To("system:dict:type:add"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 54}, ParentID: ptr.To(uint64(51)), Name: "修改字典类型", Type: "btn", Perms: ptr.To("system:dict:type:edit"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 55}, ParentID: ptr.To(uint64(51)), Name: "删除字典类型", Type: "btn", Perms: ptr.To("system:dict:type:delete"), Sort: ptr.To(4), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 56}, ParentID: ptr.To(uint64(51)), Name: "查询字典数据", Type: "btn", Perms: ptr.To("system:dict:data:query"), Sort: ptr.To(5), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 57}, ParentID: ptr.To(uint64(51)), Name: "新增字典数据", Type: "btn", Perms: ptr.To("system:dict:data:add"), Sort: ptr.To(6), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 58}, ParentID: ptr.To(uint64(51)), Name: "修改字典数据", Type: "btn", Perms: ptr.To("system:dict:data:edit"), Sort: ptr.To(7), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 59}, ParentID: ptr.To(uint64(51)), Name: "删除字典数据", Type: "btn", Perms: ptr.To("system:dict:data:delete"), Sort: ptr.To(8), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 70}, ParentID: ptr.To(uint64(51)), Name: "字典类型列表", Type: "btn", Perms: ptr.To("system:dict:type:list"), Sort: ptr.To(0), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 71}, ParentID: ptr.To(uint64(51)), Name: "字典数据列表", Type: "btn", Perms: ptr.To("system:dict:data:list"), Sort: ptr.To(9), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 参数配置 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 37}, ParentID: ptr.To(uint64(1)), Name: "参数配置", Type: "menu", Path: ptr.To("/system/config"), Component: ptr.To("system/config/index"), Perms: ptr.To("system:config:list"), Sort: ptr.To(6), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 38}, ParentID: ptr.To(uint64(37)), Name: "查询参数", Type: "btn", Perms: ptr.To("system:config:query"), Sort: ptr.To(0), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 39}, ParentID: ptr.To(uint64(37)), Name: "新增参数", Type: "btn", Perms: ptr.To("system:config:add"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 40}, ParentID: ptr.To(uint64(37)), Name: "修改参数", Type: "btn", Perms: ptr.To("system:config:edit"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 41}, ParentID: ptr.To(uint64(37)), Name: "删除参数", Type: "btn", Perms: ptr.To("system:config:delete"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 通知公告 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 42}, ParentID: ptr.To(uint64(1)), Name: "通知公告", Type: "menu", Path: ptr.To("/system/notice"), Component: ptr.To("system/notice/index"), Perms: ptr.To("system:notice:list"), Sort: ptr.To(7), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 43}, ParentID: ptr.To(uint64(42)), Name: "查询通知", Type: "btn", Perms: ptr.To("system:notice:query"), Sort: ptr.To(0), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 44}, ParentID: ptr.To(uint64(42)), Name: "新增通知", Type: "btn", Perms: ptr.To("system:notice:add"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 45}, ParentID: ptr.To(uint64(42)), Name: "修改通知", Type: "btn", Perms: ptr.To("system:notice:edit"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 46}, ParentID: ptr.To(uint64(42)), Name: "删除通知", Type: "btn", Perms: ptr.To("system:notice:delete"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 47}, ParentID: ptr.To(uint64(42)), Name: "发布通知", Type: "btn", Perms: ptr.To("system:notice:publish"), Sort: ptr.To(4), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 文件管理 ───────────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 48}, ParentID: ptr.To(uint64(1)), Name: "文件管理", Type: "menu", Path: ptr.To("/system/file"), Component: ptr.To("system/file/index"), Perms: ptr.To("system:file:list"), Sort: ptr.To(8), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 49}, ParentID: ptr.To(uint64(48)), Name: "上传文件", Type: "btn", Perms: ptr.To("system:file:upload"), Sort: ptr.To(0), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 50}, ParentID: ptr.To(uint64(48)), Name: "删除文件", Type: "btn", Perms: ptr.To("system:file:delete"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 85}, ParentID: ptr.To(uint64(48)), Name: "查询文件", Type: "btn", Perms: ptr.To("system:file:query"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	// ── 定时任务管理 ──────────────────────────────────────────
	{BaseEntity: entity.BaseEntity{ID: 60}, ParentID: ptr.To(uint64(1)), Name: "定时任务", Type: "menu", Path: ptr.To("/system/job"), Component: ptr.To("system/job/index"), Perms: ptr.To("system:job:list"), Sort: ptr.To(9), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 61}, ParentID: ptr.To(uint64(60)), Name: "查询任务", Type: "btn", Perms: ptr.To("system:job:query"), Sort: ptr.To(1), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 62}, ParentID: ptr.To(uint64(60)), Name: "新增任务", Type: "btn", Perms: ptr.To("system:job:add"), Sort: ptr.To(2), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 63}, ParentID: ptr.To(uint64(60)), Name: "修改任务", Type: "btn", Perms: ptr.To("system:job:edit"), Sort: ptr.To(3), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 64}, ParentID: ptr.To(uint64(60)), Name: "删除任务", Type: "btn", Perms: ptr.To("system:job:delete"), Sort: ptr.To(4), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 65}, ParentID: ptr.To(uint64(60)), Name: "执行任务", Type: "btn", Perms: ptr.To("system:job:execute"), Sort: ptr.To(5), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 66}, ParentID: ptr.To(uint64(60)), Name: "暂停任务", Type: "btn", Perms: ptr.To("system:job:pause"), Sort: ptr.To(6), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 67}, ParentID: ptr.To(uint64(60)), Name: "执行一次", Type: "btn", Perms: ptr.To("system:job:once"), Sort: ptr.To(7), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 68}, ParentID: ptr.To(uint64(60)), Name: "任务日志", Type: "btn", Perms: ptr.To("system:job:log:list"), Sort: ptr.To(8), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
	{BaseEntity: entity.BaseEntity{ID: 69}, ParentID: ptr.To(uint64(60)), Name: "清空日志", Type: "btn", Perms: ptr.To("system:job:log:delete"), Sort: ptr.To(9), Visible: ptr.To[int8](1), Status: ptr.To[int8](1)},
}

// seedMenusV2 seeds system menus using batch query optimization.
// It queries existing menu IDs in one batch, then only creates missing ones.
func seedMenusV2(tx *gorm.DB) error {
	var existingIDs []uint64
	if err := tx.Model(&entity.SysMenu{}).Pluck("id", &existingIDs).Error; err != nil {
		return err
	}

	existingSet := make(map[uint64]bool, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = true
	}

	var toCreate []entity.SysMenu
	for _, m := range menusV2 {
		if !existingSet[m.ID] {
			toCreate = append(toCreate, m)
		}
	}

	if len(toCreate) == 0 {
		return nil
	}

	return tx.CreateInBatches(toCreate, 100).Error
}
