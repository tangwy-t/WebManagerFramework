package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     21,
		Description: "通知接收范围:sys_notice_publish_type 值域修正为 targetType 0-3(消除 v007 旧值域新装回归)",
		Up:          migrateNoticePublishTypeV21,
	})
}

// migrateNoticePublishTypeV21 把 sys_notice_publish_type 从旧「发布类型」值域
// (1=全体 / 2=指定用户)修正为「通知接收范围」值域(0=全体成员 / 1=指定角色 /
// 2=指定部门 / 3=指定个人),与 Notices.TargetType 枚举一致。
//
// 背景:该修订此前以手工 SQL 上线(设计规格 §4),仓库内无代码痕迹,导致
// v007 种子在全新安装时仍写入旧值域,前端按 targetType 渲染时 0/3 取不到
// label 直接显示数字。本迁移将手工 SQL 落库为可复现的前向迁移:
//
//   - 已按手工脚本修订过的库:守卫检测到 value='0' 行即跳过,不覆盖运营改动;
//   - 全新安装:v007 先写旧两行,本迁移随后更名为「通知接收范围」、
//     删除旧两行并写入 0-3 四行(显式 ID 358-361,与线上手工修订一致)。
func migrateNoticePublishTypeV21(tx *gorm.DB) error {
	const code = "sys_notice_publish_type"

	// 1. 类型显示名语义修正:发布类型 → 接收范围(幂等)。
	if err := tx.Model(&entity.SysDictType{}).
		Where("code = ?", code).
		Update("name", "通知接收范围").Error; err != nil {
		return err
	}

	// 2. 守卫:值域 0 已存在(手工 SQL 已修订或本迁移已执行)则跳过数据替换。
	var revised int64
	if err := tx.Model(&entity.SysDictData{}).
		Where("type_id = (SELECT id FROM sys_dict_type WHERE code = ? LIMIT 1) AND value = '0'", code).
		Count(&revised).Error; err != nil {
		return err
	}
	if revised > 0 {
		return nil
	}

	// 3. 读取类型行获取真实 type_id(v007 种子为 201-218,但以 code 定位)。
	var dt entity.SysDictType
	if err := tx.Where(entity.SysDictType{Code: code}).First(&dt).Error; err != nil {
		return err
	}

	// 4. 物理删除旧两行(1 全体 / 2 指定用户;软删会残留旧语义数据)。
	if err := tx.Unscoped().Where("type_id = ?", dt.ID).Delete(&entity.SysDictData{}).Error; err != nil {
		return err
	}

	// 5. 写入新值域四行:与设计规格 §4 及线上手工修订(ID 358-361)完全一致。
	rows := []entity.SysDictData{
		{BaseEntity: entity.BaseEntity{ID: 358}, TypeID: dt.ID, Label: "全体成员", Value: "0", ListClass: ptr.To("primary"), IsDefault: ptr.To[int8](entity.DictDataDefaultYes), Sort: ptr.To(1), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 359}, TypeID: dt.ID, Label: "指定角色", Value: "1", ListClass: ptr.To("warning"), IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(2), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 360}, TypeID: dt.ID, Label: "指定部门", Value: "2", ListClass: ptr.To("success"), IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(3), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
		{BaseEntity: entity.BaseEntity{ID: 361}, TypeID: dt.ID, Label: "指定个人", Value: "3", ListClass: ptr.To("info"), IsDefault: ptr.To[int8](entity.DictDataDefaultNo), Sort: ptr.To(4), Status: ptr.To[int8](entity.DictDataStatusEnabled)},
	}
	for i := range rows {
		if err := tx.Where("type_id = ? AND value = ?", rows[i].TypeID, rows[i].Value).FirstOrCreate(&rows[i]).Error; err != nil {
			return err
		}
	}

	return nil
}