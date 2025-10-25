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

func (s *HistoryService) LeastSelectedProjects(limit int, alfred bool) []string {
	db := config.DataDb()

	var projects []string
	db.Model(&model.ProjectSelectLog{}).
		Select("project").
		Where(&model.ProjectSelectLog{
			Alfred: alfred,
		}).
		Group("project").
		Order("max(id) desc").
		Limit(limit).
		Find(&projects)

	return projects
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

func (s *HistoryService) LeastProjectOpenApps(project string, limit int, alfred bool) []string {
	db := config.DataDb()

	var projects []string
	db.Model(&model.ProjectOpenLog{}).
		Select("app").
		Where(&model.ProjectOpenLog{
			Project: project,
			Alfred:  alfred,
		}).
		Group("app").
		Order("max(id) desc").
		Limit(limit).
		Find(&projects)

	return projects
}
