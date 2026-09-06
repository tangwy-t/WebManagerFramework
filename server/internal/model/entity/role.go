package entity

type SysRole struct {
	BaseEntity
	Name      string    `gorm:"column:name;size:64;not null"   json:"name"`
	Code      string    `gorm:"column:code;size:64;not null;uniqueIndex:uk_role_code" json:"code"`
	DataScope *int8     `gorm:"column:data_scope;default:1"    json:"dataScope"`
	Sort      *int      `gorm:"column:sort;default:0"          json:"sort"`
	Status    *int8     `gorm:"column:status;default:1"        json:"status"`
	Remark    *string   `gorm:"column:remark;size:512"         json:"remark"`
	Menus     []SysMenu `gorm:"many2many:sys_role_menu;joinForeignKey:role_id;joinReferences:menu_id"                             json:"menus,omitempty"`
	Depts     []SysDept `gorm:"many2many:sys_role_dept;joinForeignKey:role_id;joinReferences:dept_id" json:"depts,omitempty"`
}

// Deprecated: Use dict service FindDataByCode(ctx, "sys_role_status") instead.
// Kept for backward compatibility.
const (
	RoleStatusEnabled  int8 = 1
	RoleStatusDisabled int8 = 0
)

func (SysRole) TableName() string { return "sys_role" }
