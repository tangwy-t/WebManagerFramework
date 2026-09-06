package migrations

import (
	"gorm.io/gorm"

	"github.com/tangwy-t/webmanager-server/internal/model/entity"
	"github.com/tangwy-t/webmanager-server/internal/pkg/migration"
	"github.com/tangwy-t/webmanager-server/internal/pkg/ptr"
)

func init() {
	migration.Register(migration.Migration{
		Version:     8,
		Description: "添加配置/字典同步任务（run_at_startup）",
		Up:          seedConfigDictSyncV8,
	})
}

func seedConfigDictSyncV8(tx *gorm.DB) error {
	// 1. Add run_at_startup column (ignore error if already exists)
	if err := tx.Exec("ALTER TABLE sys_job ADD COLUMN run_at_startup TINYINT(1) DEFAULT 0").Error; err != nil {
		// Column may already exist; ignore the error
		_ = err
	}

	// 2. Seed config:sync and dict:sync jobs
	jobs := []entity.SysJob{
		{
			BaseEntity:     entity.BaseEntity{ID: 100004},
			Name:           "配置全量同步",
			JobGroup:       "system",
			CronExpression: "0 0 2 * * *",
			InvokeTarget:   "config-cache-sync",
			Concurrent:     ptr.To[int8](entity.JobConcurrentAllowed),
			Status:         ptr.To[int8](entity.JobStatusEnabled),
			RunAtStartup:   ptr.To[int8](entity.JobRunAtStartupYes),
		},
		{
			BaseEntity:     entity.BaseEntity{ID: 100005},
			Name:           "字典全量同步",
			JobGroup:       "system",
			CronExpression: "0 0 3 * * *",
			InvokeTarget:   "dict-cache-sync",
			Concurrent:     ptr.To[int8](entity.JobConcurrentAllowed),
			Status:         ptr.To[int8](entity.JobStatusEnabled),
			RunAtStartup:   ptr.To[int8](entity.JobRunAtStartupYes),
		},
	}

	for _, j := range jobs {
		if err := tx.Where("invoke_target = ?", j.InvokeTarget).FirstOrCreate(&j).Error; err != nil {
			return err
		}
	}
	return nil
}
