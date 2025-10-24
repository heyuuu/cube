package model

import (
	"gorm.io/gorm"
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
