package repository

import (
	"context"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"

	"gorm.io/gorm"
)

type MenuRepo struct {
	db *gorm.DB
}

// NewMenuRepository returns a MenuRepository backed by the given *gorm.DB.
func NewMenuRepository(db *gorm.DB) *MenuRepo {
	return &MenuRepo{db: db}
}

func (r *MenuRepo) FindAll(ctx context.Context) ([]entity.SysMenu, error) {
	var menus []entity.SysMenu
	err := r.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&menus).Error
	return menus, err
}

func (r *MenuRepo) FindByID(ctx context.Context, id uint64) (*entity.SysMenu, error) {
	var menu entity.SysMenu
	err := r.db.WithContext(ctx).First(&menu, id).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *MenuRepo) Create(ctx context.Context, menu *entity.SysMenu) error {
	return r.db.WithContext(ctx).Create(menu).Error
}

func (r *MenuRepo) Update(ctx context.Context, menu *entity.SysMenu) error {
	return r.db.WithContext(ctx).Model(menu).Updates(menu).Error
}

func (r *MenuRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&entity.SysMenu{}, id).Error
}

func (r *MenuRepo) HasChildren(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.SysMenu{}).Where("parent_id = ?", id).Count(&count).Error
	return count > 0, err
}
