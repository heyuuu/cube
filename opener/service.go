package opener

import (
	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/util/matcher"
	"github.com/heyuuu/cube/util/slicekit"
)

type Service struct {
	apps []*Opener
}

func NewService(conf config.Config) *Service {
	// 读取配置
	apps := slicekit.Map(conf.Openers, NewOpener)

	return &Service{
		apps: apps,
	}
}

func (s *Service) Openers() []*Opener {
	return s.apps
}

func (s *Service) FindByName(name string) *Opener {
	for _, app := range s.apps {
		if app.Name() == name {
			return app
		}
	}

	return nil
}

func (s *Service) Search(query string) []*Opener {
	if len(query) == 0 || len(s.apps) == 0 {
		return s.apps
	}

	// match
	m := matcher.NewKeywordMatcher(s.apps, func(app *Opener) string {
		return app.Name()
	}, nil)
	return m.Match(query)
}
