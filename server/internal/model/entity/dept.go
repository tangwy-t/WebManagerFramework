package entity

import "github.com/tangwy-t/webmanager-server/internal/pkg/datascope/rule"

type SysDept struct {
	BaseEntity
	ParentID  *uint64    `gorm:"column:parent_id;default:0"     json:"parentId,string"`
	Ancestors string     `gorm:"column:ancestors;size:512;default:''" json:"ancestors"`
	Name      string     `gorm:"column:name;size:64;not null"   json:"name"`
	Sort      *int       `gorm:"column:sort;default:0"          json:"sort"`
	Leader    *string    `gorm:"column:leader;size:64"          json:"leader"`
	Phone     *string    `gorm:"column:phone;size:20"           json:"phone"`
	Email     *string    `gorm:"column:email;size:128"          json:"email"`
	Status    *int8      `gorm:"column:status;default:1"        json:"status"`
	Children  []*SysDept `gorm:"-"                              json:"children,omitempty"`
}

// Kept for backward compatibility.
const (
	DeptStatusEnabled  int8 = 1
	DeptStatusDisabled int8 = 0
)

func (SysDept) TableName() string { return "sys_dept" }

// DataScopeRules implements rule.DataScopeable.
func (SysDept) DataScopeRules() []rule.ScopeRule {
	return []rule.ScopeRule{
		{Column: "id", DimensionType: "dept"},
		{Column: "id", DimensionType: "self"},
	}
}
