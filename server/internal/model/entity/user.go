package entity

import (
	"time"

	"github.com/tangwy-t/webmanager-server/internal/pkg/datascope/rule"
)

type SysUser struct {
	BaseEntity
	Username      string     `gorm:"column:username;size:64;not null;uniqueIndex:uk_user_username" json:"username"`
	Password      string     `gorm:"column:password;size:255;not null"      json:"-"`
	PasswordSalt  *string    `gorm:"column:password_salt;size:64"           json:"-"`
	RealName      *string    `gorm:"column:real_name;size:64"               json:"realName"`
	Email         *string    `gorm:"column:email;size:128"                  json:"email"`
	Phone         *string    `gorm:"column:phone;size:20"                   json:"phone"`
	Avatar        *string    `gorm:"column:avatar;size:512"                 json:"avatar"`
	DeptID        *uint64    `gorm:"column:dept_id;index:idx_user_dept_id"  json:"deptId,string"`
	Status        *int8      `gorm:"column:status;default:1"                json:"status"`
	Remark        *string    `gorm:"column:remark;size:512"                 json:"remark"`
	LastLoginTime *time.Time `gorm:"column:last_login_time"                 json:"lastLoginTime"`
	LastLoginIP   *string    `gorm:"column:last_login_ip;size:64"           json:"lastLoginIp"`
	Dept          *SysDept   `gorm:"foreignKey:DeptID"                      json:"dept,omitempty"`
	Roles         []SysRole  `gorm:"many2many:sys_user_role;joinForeignKey:user_id;joinReferences:role_id"                json:"roles,omitempty"`
}

// Kept for backward compatibility.
const (
	UserStatusEnabled  int8 = 1
	UserStatusDisabled int8 = 0
)

func (SysUser) TableName() string { return "sys_user" }

// DataScopeRules implements rule.DataScopeable.
func (SysUser) DataScopeRules() []rule.ScopeRule {
	return []rule.ScopeRule{
		{Column: "dept_id", DimensionType: "dept"},
		{Column: "id", DimensionType: "self"},
	}
}
