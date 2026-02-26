package schedule

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *gorm.DB { return r.db }

func (r *Repository) GetSystemState() (*SystemState, error) {
	var st SystemState
	if err := r.db.First(&st, "id = 1").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			st = SystemState{ID: 1, ScheduleVersion: time.Now().UTC()}
			if err2 := r.db.Create(&st).Error; err2 != nil {
				return nil, fmt.Errorf("init system_state: %w", err2)
			}
			return &st, nil
		}
		return nil, err
	}
	return &st, nil
}

func (r *Repository) BumpScheduleVersion() error {
	return r.db.Model(&SystemState{}).Where("id = 1").Update("schedule_version", time.Now().UTC()).Error
}
