package services

import (
	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/entities"
	"github.com/heyuuu/cube/util/matcher"
	"github.com/heyuuu/cube/util/slicekit"
)

type ApplicationService struct {
	apps []*entities.Application
}

func NewApplicationService(conf config.Config) *ApplicationService {
	// 读取配置
	apps := slicekit.Map(conf.Applications, entities.NewApplication)

	return &ApplicationService{
		apps: apps,
	}
}

func (s *ApplicationService) Apps() []*entities.Application {
	return s.apps
}

func (s *ApplicationService) FindByName(name string) *entities.Application {
	for _, app := range s.apps {
		if app.Name() == name {
			return app
		}
	}

	return nil
}

func (s *ApplicationService) Search(query string) []*entities.Application {
	if len(query) == 0 || len(s.apps) == 0 {
		return s.apps
	}

	// match
	m := matcher.NewKeywordMatcher(s.apps, func(app *entities.Application) string {
		return app.Name()
	}, nil)
	return m.Match(query)
}
