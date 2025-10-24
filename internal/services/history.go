package services

import (
	"github.com/heyuuu/go-cube/internal/config"
	"github.com/heyuuu/go-cube/internal/model"
	"gorm.io/gorm"
)

type HistoryService struct {
	db *gorm.DB
}

func NewHistoryService() *HistoryService {
	return &HistoryService{}
}

func (s *HistoryService) AddProjectSelectLog(project string, alfred bool) error {
	db := config.DataDb()

	m := &model.ProjectSelectLog{
		Project: project,
		Alfred:  alfred,
	}
	db.Create(m)
	return nil
}

func (s *HistoryService) AddProjectOpenLog(project string, app string, alfred bool) error {
	db := config.DataDb()

	m := &model.ProjectOpenLog{
		Project: project,
		App:     app,
		Alfred:  alfred,
	}
	db.Create(m)
	return nil
}
