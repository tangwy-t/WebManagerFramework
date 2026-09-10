package repository

import (
	"context"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

// NewUserRepository returns a UserRepository backed by the given *gorm.DB.
func NewUserRepository(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) applyUserFilters(db *gorm.DB, query *request.UserQuery) *gorm.DB {
	if query.Username != "" {
		db = db.Where("username LIKE ?", "%"+query.Username+"%")
	}
	if query.RealName != "" {
		db = db.Where("real_name LIKE ?", "%"+query.RealName+"%")
	}
	if query.Phone != "" {
		db = db.Where("phone LIKE ?", "%"+query.Phone+"%")
	}
	if query.Email != "" {
		db = db.Where("email LIKE ?", "%"+query.Email+"%")
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	if query.DeptID != nil {
		db = db.Where("dept_id = ?", *query.DeptID)
	}
	if query.RoleID != nil {
		// 过滤已分配该角色的用户:不用 JOIN,交给子查询,避免与 Roles Preload 的
		// many2many JOIN 产生笛卡尔积。
		db = db.Where("id IN (SELECT user_id FROM sys_user_role WHERE role_id = ?)", *query.RoleID)
	}
	if query.CreatedAtStart != "" {
		db = db.Where("created_at >= ?", query.CreatedAtStart+" 00:00:00")
	}
	if query.CreatedAtEnd != "" {
		db = db.Where("created_at <= ?", query.CreatedAtEnd+" 23:59:59")
	}
	return db
}

func (r *UserRepo) FindPage(ctx context.Context, query *request.UserQuery) ([]entity.SysUser, int64, error) {
	// Count without Preload to avoid unnecessary JOIN overhead.
	var total int64
	countDB := r.applyUserFilters(r.db.WithContext(ctx).Model(&entity.SysUser{}), query)
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataDB := r.applyUserFilters(r.db.WithContext(ctx).Model(&entity.SysUser{}), query)
	dataDB = dataDB.Preload("Dept").Preload("Roles")
	var users []entity.SysUser
	err := dataDB.Offset(query.Offset()).Limit(query.GetPageSize()).Order("id DESC").Find(&users).Error
	return users, total, err
}

func (r *UserRepo) FindByID(ctx context.Context, id uint64) (*entity.SysUser, error) {
	var user entity.SysUser
	err := r.db.WithContext(ctx).Preload("Dept").Preload("Roles").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*entity.SysUser, error) {
	var user entity.SysUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateWithRoles inserts a user together with their role assignments.
//
// Join rows are created explicitly (see ReplaceRoles) rather than through
// Association("Roles").Append: association-created join rows never received a
// snowflake ID, so any user created with two or more roles silently kept only
// the first assignment.
func (r *UserRepo) CreateWithRoles(ctx context.Context, user *entity.SysUser, roleIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Create(user).Error; err != nil {
			return err
		}
		rows := buildUserRoleRows(user.ID, roleIDs)
		if len(rows) == 0 {
			return nil
		}
		return tx.WithContext(ctx).Create(&rows).Error
	})
}

func (r *UserRepo) Update(ctx context.Context, user *entity.SysUser) error {
	// status is omitted so the info-update endpoint never touches the status
	// column — status changes must go through Enable/Disable (self-guarded).
	// FindByID preloads a non-nil status; without this Omit, Updates would
	// rewrite it on every UpdateUserInfo, racing concurrent Disable/Enable.
	return r.db.WithContext(ctx).Model(user).
		Omit("password", "last_login_time", "last_login_ip", "status").
		Updates(user).Error
}

// ReplaceRoles atomically swaps a user's role set for exactly roleIDs.
//
// The join rows are written through the SysUserRole entity instead of GORM's
// Association("Roles").Replace. Association writes go through an internal
// join-table schema that the global id:generate callback does not cover, so
// every inserted row kept ID = 0. Because id is the primary key, only the
// first row could ever be stored: a second role silently died on a duplicate
// key, and Association.Replace does not surface that error. Writing the rows
// explicitly lets the callback assign a snowflake ID to each one and makes
// insert failures visible to the caller.
func (r *UserRepo) ReplaceRoles(ctx context.Context, id uint64, roleIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).
			Where("user_id = ?", id).
			Delete(&entity.SysUserRole{}).Error; err != nil {
			return err
		}
		rows := buildUserRoleRows(id, roleIDs)
		if len(rows) == 0 {
			return nil
		}
		return tx.WithContext(ctx).Create(&rows).Error
	})
}

// buildUserRoleRows converts role IDs into join rows, de-duplicating role IDs
// so a repeated ID in the request cannot violate the (user_id, role_id)
// unique index.
func buildUserRoleRows(userID uint64, roleIDs []uint64) []entity.SysUserRole {
	if len(roleIDs) == 0 {
		return nil
	}
	seen := make(map[uint64]struct{}, len(roleIDs))
	rows := make([]entity.SysUserRole, 0, len(roleIDs))
	for _, rid := range roleIDs {
		if _, ok := seen[rid]; ok {
			continue
		}
		seen[rid] = struct{}{}
		rows = append(rows, entity.SysUserRole{UserID: userID, RoleID: rid})
	}
	return rows
}

func (r *UserRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&entity.SysUser{}, id).Error
}

func (r *UserRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	return r.db.WithContext(ctx).Model(&entity.SysUser{}).Where("id = ?", id).
		Update("status", status).Error
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uint64, password string, salt *string) error {
	updates := map[string]interface{}{"password": password}
	if salt != nil {
		updates["password_salt"] = *salt
	}
	return r.db.WithContext(ctx).Model(&entity.SysUser{}).Where("id = ?", id).
		Updates(updates).Error
}

// FindRoleCodeByID returns the role's code (used by role-member guards:
// the built-in admin role's membership is not modifiable via the role page).
func (r *UserRepo) FindRoleCodeByID(ctx context.Context, roleID uint64) (string, error) {
	var code string
	err := r.db.WithContext(ctx).Model(&entity.SysRole{}).
		Where("id = ?", roleID).Pluck("code", &code).Error
	return code, err
}

// AddUsersToRole inserts role-user join rows for userIDs that are not yet
// assigned, skipping already-assigned ones (idempotent for repeated adds).
// Snowflake IDs are generated by the global GORM id:generate callback.
func (r *UserRepo) AddUsersToRole(ctx context.Context, roleID uint64, userIDs []uint64) error {
	if len(userIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing []uint64
		if err := tx.Model(&entity.SysUserRole{}).
			Where("role_id = ? AND user_id IN ?", roleID, userIDs).
			Pluck("user_id", &existing).Error; err != nil {
			return err
		}
		seen := make(map[uint64]struct{}, len(existing))
		for _, uid := range existing {
			seen[uid] = struct{}{}
		}
		rows := make([]entity.SysUserRole, 0, len(userIDs))
		for _, uid := range userIDs {
			if _, ok := seen[uid]; ok {
				continue
			}
			seen[uid] = struct{}{}
			rows = append(rows, entity.SysUserRole{RoleID: roleID, UserID: uid})
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.WithContext(ctx).Create(&rows).Error
	})
}

// RemoveUsersFromRole deletes role-user join rows for the given users only.
func (r *UserRepo) RemoveUsersFromRole(ctx context.Context, roleID uint64, userIDs []uint64) error {
	if len(userIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("role_id = ? AND user_id IN ?", roleID, userIDs).
		Delete(&entity.SysUserRole{}).Error
}
