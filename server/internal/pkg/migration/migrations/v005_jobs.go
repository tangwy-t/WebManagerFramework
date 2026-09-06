package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     5,
		Description: "种子清理任务",
		Up:          seedJobsV5,
	})
}

func seedJobsV5(tx *gorm.DB) error {
	jobs := []entity.SysJob{
		{
			BaseEntity:     entity.BaseEntity{ID: 100001},
			Name:           "操作日志清理",
			JobGroup:       "system",
			CronExpression: "0 0 3 * * *",
			InvokeTarget:   "op-log-cleanup",
			Concurrent:     ptr.To[int8](entity.JobConcurrentAllowed),
			Status:         ptr.To[int8](entity.JobStatusEnabled),
		},
		{
			BaseEntity:     entity.BaseEntity{ID: 100002},
			Name:           "登录日志清理",
			JobGroup:       "system",
			CronExpression: "0 0 3 * * *",
			InvokeTarget:   "login-log-cleanup",
			Concurrent:     ptr.To[int8](entity.JobConcurrentAllowed),
			Status:         ptr.To[int8](entity.JobStatusEnabled),
		},
		{
			BaseEntity:     entity.BaseEntity{ID: 100003},
			Name:           "任务日志清理",
			JobGroup:       "system",
			CronExpression: "0 0 3 * * *",
			InvokeTarget:   "job-log-cleanup",
			Concurrent:     ptr.To[int8](entity.JobConcurrentAllowed),
			Status:         ptr.To[int8](entity.JobStatusEnabled),
		},
	}

	for _, j := range jobs {
		if err := tx.Where("name = ? AND job_group = ?", j.Name, j.JobGroup).FirstOrCreate(&j).Error; err != nil {
			return err
		}
	}
	return nil
}
