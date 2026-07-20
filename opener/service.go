package opener

import (
	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/util/fuzzy"
	"github.com/heyuuu/cube/util/slicekit"
)

type Service struct {
	openers []*Opener
}

func NewService(conf config.Config) *Service {
	// 读取配置
	openers := slicekit.Map(conf.Openers, NewOpener)

	return &Service{
		openers: openers,
	}
}

func (s *Service) Openers() []*Opener {
	return s.openers
}

func (s *Service) FindByName(name string) *Opener {
	for _, app := range s.openers {
		if app.Name() == name {
			return app
		}
	}

	return nil
}

func (s *Service) Search(query string) []*Opener {
	return fuzzy.MatchBy(query, s.openers, (*Opener).Name, nil)
}
