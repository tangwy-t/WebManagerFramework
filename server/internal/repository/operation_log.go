package repository

import (
	"context"
	"time"

	"github.com/tangwy-t/webmanager-server/internal/model/dto/request"
	"github.com/tangwy-t/webmanager-server/internal/model/entity"

	"gorm.io/gorm"
)

type OperationLogRepo struct {
	db *gorm.DB
}

// NewOperationLogRepository returns an OperationLogRepository backed by the given *gorm.DB.
func NewOperationLogRepository(db *gorm.DB) *OperationLogRepo {
	return &OperationLogRepo{db: db}
}

func (r *OperationLogRepo) applyFilters(db *gorm.DB, query *request.OperationLogQuery) *gorm.DB {
	if query.Username != "" {
		db = db.Where("username LIKE ?", "%"+query.Username+"%")
	}
	if query.Module != "" {
		db = db.Where("module = ?", query.Module)
	}
	if query.OperationType != "" {
		db = db.Where("operation_type = ?", query.OperationType)
	}
	if query.Code != nil {
		db = db.Where("code = ?", *query.Code)
	}
	if query.StartTime != "" {
		if t, err := time.Parse("2006-01-02", query.StartTime); err == nil {
			db = db.Where("oper_time >= ?", t)
		}
	}
	if query.EndTime != "" {
		if t, err := time.Parse("2006-01-02", query.EndTime); err == nil {
			db = db.Where("oper_time < ?", t.Add(24*time.Hour))
		}
	}
	return db
}

func (r *OperationLogRepo) FindPage(ctx context.Context, query *request.OperationLogQuery) ([]entity.SysOperationLog, int64, error) {
	var total int64
	countDB := r.applyFilters(r.db.WithContext(ctx).Model(&entity.SysOperationLog{}), query)
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []entity.SysOperationLog
	dataDB := r.applyFilters(r.db.WithContext(ctx).Model(&entity.SysOperationLog{}), query)
	err := dataDB.Offset(query.Offset()).Limit(query.GetPageSize()).Order("id DESC").Find(&logs).Error
	return logs, total, err
}

func (r *OperationLogRepo) Create(ctx context.Context, log *entity.SysOperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *OperationLogRepo) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("oper_time < ?", before).Delete(&entity.SysOperationLog{})
	return result.RowsAffected, result.Error
}
