package history

import (
	"gorm.io/gorm"

	"github.com/heyuuu/cube/db"
)

type ProjectSelectLog struct {
	gorm.Model
	Project string `gorm:"project"`
	Alfred  bool   `gorm:"alfred"`
}

type ProjectOpenLog struct {
	gorm.Model
	Project string `gorm:"project"`
	App     string `gorm:"app"`
	Alfred  bool   `gorm:"alfred"`
}

type HistoryService struct {
	db *gorm.DB
}

func NewHistoryService() *HistoryService {
	return &HistoryService{}
}

func (s *HistoryService) AddProjectSelectLog(project string, alfred bool) error {
	db := db.DataDb()

	m := &ProjectSelectLog{
		Project: project,
		Alfred:  alfred,
	}
	db.Create(m)
	return nil
}

func (s *HistoryService) LeastSelectedProjects(limit int, alfred bool) []string {
	db := db.DataDb()

	var projects []string
	db.Model(&ProjectSelectLog{}).
		Select("project").
		Where(&ProjectSelectLog{
			Alfred: alfred,
		}).
		Group("project").
		Order("max(id) desc").
		Limit(limit).
		Find(&projects)

	return projects
}

func (s *HistoryService) AddProjectOpenLog(project string, app string, alfred bool) error {
	db := db.DataDb()

	m := &ProjectOpenLog{
		Project: project,
		App:     app,
		Alfred:  alfred,
	}
	db.Create(m)
	return nil
}

func (s *HistoryService) LeastProjectOpenApps(project string, limit int, alfred bool) []string {
	db := db.DataDb()

	var projects []string
	db.Model(&ProjectOpenLog{}).
		Select("app").
		Where(&ProjectOpenLog{
			Project: project,
			Alfred:  alfred,
		}).
		Group("app").
		Order("max(id) desc").
		Limit(limit).
		Find(&projects)

	return projects
}
