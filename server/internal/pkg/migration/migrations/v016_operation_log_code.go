package migrations

import (
	"encoding/json"

	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     16,
		Description: "操作日志结果码:新增字典 sys_opt_result_code(全部业务码) + 历史失败日志 code 回填",
		Up:          migrateOptResultCodeV16,
	})
}

// optResultCodes 操作日志结果码字典:value 与 apperror 包的业务码一一对应。
var optResultCodes = []struct {
	value, label, listClass string
	sort                    int
}{
	{"0", "成功", "success", 1},
	{"10001", "未登录/令牌缺失", "warning", 2},
	{"10002", "无操作权限", "warning", 3},
	{"10003", "令牌已过期", "warning", 4},
	{"10004", "需要验证码", "warning", 5},
	{"10005", "验证码错误", "danger", 6},
	{"10006", "验证码已过期", "warning", 7},
	{"10007", "请求过于频繁", "danger", 8},
	{"10008", "账号已锁定", "danger", 9},
	{"40000", "参数错误", "danger", 10},
	{"40400", "资源不存在", "danger", 11},
	{"40900", "操作冲突", "danger", 12},
	{"50000", "服务器内部错误", "danger", 13},
}

func migrateOptResultCodeV16(tx *gorm.DB) error {
	// 1. 新字典类型 sys_opt_result_code(ID 220)
	codeType := entity.SysDictType{
		BaseEntity: entity.BaseEntity{ID: 220},
		Code:       "sys_opt_result_code",
		Name:       "操作日志结果码",
		Status:     ptr.To[int8](entity.DictTypeStatusEnabled),
	}
	if err := tx.Where(entity.SysDictType{Code: codeType.Code}).FirstOrCreate(&codeType).Error; err != nil {
		return err
	}

	// 2. 数据项(ID 345-357),全部业务码定义到字典
	for i, c := range optResultCodes {
		dd := entity.SysDictData{
			BaseEntity: entity.BaseEntity{ID: uint64(345 + i)},
			TypeID:     codeType.ID,
			Label:      c.label,
			Value:      c.value,
			ListClass:  ptr.To(c.listClass),
			Sort:       ptr.To(c.sort),
			Status:     ptr.To[int8](entity.DictDataStatusEnabled),
		}
		if err := tx.Where("type_id = ? AND value = ?", dd.TypeID, dd.Value).FirstOrCreate(&dd).Error; err != nil {
			return err
		}
	}

	// 3. 历史失败日志 code 回填:旧 status 时代失败行(error_msg 非空且 code 仍为 0)
	//    从 error_msg 的信封 JSON 解析真实业务码,解析不了按 50000 记。
	var legacy []entity.SysOperationLog
	if err := tx.Where("code = ? AND error_msg IS NOT NULL AND error_msg <> ''", apperror.CodeOK).
		Find(&legacy).Error; err != nil {
		return err
	}
	for i := range legacy {
		code := apperror.CodeInternal
		var env struct {
			Code *int `json:"code"`
		}
		if legacy[i].ErrorMsg != nil {
			if err := json.Unmarshal([]byte(*legacy[i].ErrorMsg), &env); err == nil && env.Code != nil && *env.Code != apperror.CodeOK {
				code = *env.Code
			}
		}
		if err := tx.Model(&entity.SysOperationLog{}).Where("id = ?", legacy[i].ID).Update("code", code).Error; err != nil {
			return err
		}
	}

	return nil
}