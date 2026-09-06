package entity

type SysRoleMenu struct {
	ID     uint64 `gorm:"column:id;primaryKey;autoIncrement:false" json:"id,string"`
	RoleID uint64 `gorm:"column:role_id;not null"                  json:"roleId,string"`
	MenuID uint64 `gorm:"column:menu_id;not null"                  json:"menuId,string"`
}

func (SysRoleMenu) TableName() string { return "sys_role_menu" }
