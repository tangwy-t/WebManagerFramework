package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     7,
		Description: "写入字典种子数据（18个字典类型）",
		Up:          seedDictV7,
	})
}

func seedDictV7(tx *gorm.DB) error {
	// 1. 创建18个字典类型
	types := []entity.SysDictType{
		{BaseEntity: entity.BaseEntity{ID: 201}, Code: "sys_normal_disable", Name: "系统开关", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 202}, Code: "sys_show_hide", Name: "显示状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 203}, Code: "sys_yes_no", Name: "系统是否", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 204}, Code: "sys_notice_type", Name: "通知类型", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 205}, Code: "sys_notice_status", Name: "通知状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 206}, Code: "sys_notice_priority", Name: "通知优先级", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 207}, Code: "sys_notice_publish_type", Name: "通知发布类型", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 208}, Code: "sys_notice_read_status", Name: "通知阅读状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 209}, Code: "sys_job_status", Name: "任务状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 210}, Code: "sys_job_concurrent", Name: "任务并发", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 211}, Code: "sys_job_log_trigger", Name: "任务日志触发类型", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 212}, Code: "sys_job_log_status", Name: "任务日志状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 213}, Code: "sys_config_status", Name: "参数配置状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 214}, Code: "sys_dict_status", Name: "字典数据状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 215}, Code: "sys_menu_status", Name: "菜单状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 216}, Code: "sys_role_status", Name: "角色状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 217}, Code: "sys_dept_status", Name: "部门状态", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 218}, Code: "sys_job_run_at_startup", Name: "任务启动执行", Status: ptr.To[int8](entity.DictTypeStatusEnabled)},
	}

	for _, dt := range types {
		if err := tx.Where(entity.SysDictType{Code: dt.Code}).FirstOrCreate(&dt).Error; err != nil {
			return err
		}
	}

	// 2. 创建字典数据条目
	data := []entity.SysDictData{
		// sys_normal_disable (201)
		{BaseEntity: entity.BaseEntity{ID: 301}, TypeID: 201, Label: "正常", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 302}, TypeID: 201, Label: "停用", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_show_hide (202)
		{BaseEntity: entity.BaseEntity{ID: 303}, TypeID: 202, Label: "显示", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 304}, TypeID: 202, Label: "隐藏", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_yes_no (203)
		{BaseEntity: entity.BaseEntity{ID: 305}, TypeID: 203, Label: "是", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 306}, TypeID: 203, Label: "否", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_notice_type (204)
		{BaseEntity: entity.BaseEntity{ID: 307}, TypeID: 204, Label: "通知", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 308}, TypeID: 204, Label: "公告", Value: "2", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_notice_status (205)
		{BaseEntity: entity.BaseEntity{ID: 309}, TypeID: 205, Label: "草稿", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 310}, TypeID: 205, Label: "已发布", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 311}, TypeID: 205, Label: "已撤回", Value: "2", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(3), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_notice_priority (206)
		{BaseEntity: entity.BaseEntity{ID: 312}, TypeID: 206, Label: "普通", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 313}, TypeID: 206, Label: "重要", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 314}, TypeID: 206, Label: "紧急", Value: "2", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(3), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_notice_publish_type (207)
		{BaseEntity: entity.BaseEntity{ID: 315}, TypeID: 207, Label: "全体", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 316}, TypeID: 207, Label: "指定用户", Value: "2", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_notice_read_status (208)
		{BaseEntity: entity.BaseEntity{ID: 317}, TypeID: 208, Label: "未读", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 318}, TypeID: 208, Label: "已读", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_job_status (209)
		{BaseEntity: entity.BaseEntity{ID: 319}, TypeID: 209, Label: "运行中", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 320}, TypeID: 209, Label: "暂停", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_job_concurrent (210)
		{BaseEntity: entity.BaseEntity{ID: 321}, TypeID: 210, Label: "允许", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 322}, TypeID: 210, Label: "禁止", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_job_log_trigger (211)
		{BaseEntity: entity.BaseEntity{ID: 323}, TypeID: 211, Label: "定时触发", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 324}, TypeID: 211, Label: "手动执行", Value: "2", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_job_log_status (212)
		{BaseEntity: entity.BaseEntity{ID: 325}, TypeID: 212, Label: "执行中", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 326}, TypeID: 212, Label: "成功", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 327}, TypeID: 212, Label: "失败", Value: "2", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(3), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_config_status (213)
		{BaseEntity: entity.BaseEntity{ID: 328}, TypeID: 213, Label: "启用", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 329}, TypeID: 213, Label: "禁用", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_dict_status (214)
		{BaseEntity: entity.BaseEntity{ID: 330}, TypeID: 214, Label: "启用", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 331}, TypeID: 214, Label: "禁用", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_menu_status (215)
		{BaseEntity: entity.BaseEntity{ID: 332}, TypeID: 215, Label: "启用", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 333}, TypeID: 215, Label: "禁用", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_role_status (216)
		{BaseEntity: entity.BaseEntity{ID: 334}, TypeID: 216, Label: "启用", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 335}, TypeID: 216, Label: "禁用", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_dept_status (217)
		{BaseEntity: entity.BaseEntity{ID: 336}, TypeID: 217, Label: "启用", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 337}, TypeID: 217, Label: "禁用", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		// sys_job_run_at_startup (218)
		{BaseEntity: entity.BaseEntity{ID: 338}, TypeID: 218, Label: "是", Value: "1", IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 339}, TypeID: 218, Label: "否", Value: "0", IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
	}

	for _, dd := range data {
		if err := tx.Where("type_id = ? AND value = ?", dd.TypeID, dd.Value).FirstOrCreate(&dd).Error; err != nil {
			return err
		}
	}

	return nil
}
