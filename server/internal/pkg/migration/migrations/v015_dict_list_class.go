package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     15,
		Description: "字典样式：sys_list_class 类型 + list_class 字段数据（新建类型及 39 项样式预填）",
		Up:          seedDictListClassV15,
	})
}

// dictListClass 现有字典项的样式预填表：按 (typeID, value) 定位。
// 语义规则：正向→success，负向/禁用→danger，中性→info，警示→warning，选项型→primary。
var dictListClass = []struct {
	typeID           uint64
	value, listClass string
}{
	{201, "1", "success"}, // 正常
	{201, "0", "danger"},  // 停用
	{202, "1", "primary"}, // 显示
	{202, "0", "info"},    // 隐藏
	{203, "1", "primary"}, // 是
	{203, "0", "info"},    // 否
	{204, "1", "primary"}, // 通知
	{204, "2", "warning"}, // 公告
	{205, "0", "info"},    // 草稿
	{205, "1", "success"}, // 已发布
	{205, "2", "warning"}, // 已撤回
	{206, "0", "info"},    // 普通
	{206, "1", "warning"}, // 重要
	{206, "2", "danger"},  // 紧急
	{207, "1", "primary"}, // 全体
	{207, "2", "info"},    // 指定用户
	{208, "0", "warning"}, // 未读
	{208, "1", "info"},    // 已读
	{209, "1", "success"}, // 运行中
	{209, "0", "danger"},  // 暂停
	{210, "1", "success"}, // 允许
	{210, "0", "danger"},  // 禁止
	{211, "1", "primary"}, // 定时触发
	{211, "2", "info"},    // 手动执行
	{212, "0", "warning"}, // 执行中
	{212, "1", "success"}, // 成功
	{212, "2", "danger"},  // 失败
	{213, "1", "success"}, // 启用
	{213, "0", "danger"},  // 禁用
	{214, "1", "success"}, // 启用
	{214, "0", "danger"},  // 禁用
	{215, "1", "success"}, // 启用
	{215, "0", "danger"},  // 禁用
	{216, "1", "success"}, // 启用
	{216, "0", "danger"},  // 禁用
	{217, "1", "success"}, // 启用
	{217, "0", "danger"},  // 禁用
	{218, "1", "primary"}, // 是
	{218, "0", "info"},    // 否
}

func seedDictListClassV15(tx *gorm.DB) error {
	// 1. 新字典类型 sys_list_class（ID 219）
	styleType := entity.SysDictType{
		BaseEntity: entity.BaseEntity{ID: 219},
		Code:       "sys_list_class",
		Name:       "字典样式",
		Status:     ptr.To[int8](entity.DictTypeStatusEnabled),
	}
	if err := tx.Where(entity.SysDictType{Code: styleType.Code}).FirstOrCreate(&styleType).Error; err != nil {
		return err
	}

	// 2. 5 个数据项（ID 340-344），每项 list_class = 自身 value（自描述）
	items := []entity.SysDictData{
		{BaseEntity: entity.BaseEntity{ID: 340}, TypeID: 219, Label: "主要", Value: "primary", ListClass: ptr.To("primary"), IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 341}, TypeID: 219, Label: "成功", Value: "success", ListClass: ptr.To("success"), IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 342}, TypeID: 219, Label: "信息", Value: "info", ListClass: ptr.To("info"), IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(3), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 343}, TypeID: 219, Label: "警告", Value: "warning", ListClass: ptr.To("warning"), IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(4), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 344}, TypeID: 219, Label: "危险", Value: "danger", ListClass: ptr.To("danger"), IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(5), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
	}
	for _, dd := range items {
		if err := tx.Where("type_id = ? AND value = ?", dd.TypeID, dd.Value).FirstOrCreate(&dd).Error; err != nil {
			return err
		}
	}

	// 3. 现有 39 项样式预填（UPDATE 幂等）
	for _, c := range dictListClass {
		if err := tx.Model(&entity.SysDictData{}).
			Where("type_id = ? AND value = ?", c.typeID, c.value).
			Update("list_class", c.listClass).Error; err != nil {
			return err
		}
	}

	return nil
}
