package project

import (
	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/util/slicekit"
)

type RemoteService struct {
	remotes []*Remote
}

func NewRemoteService(conf config.Config) *RemoteService {
	remotes := slicekit.Map(conf.Remotes, NewRemote)

	return &RemoteService{
		remotes: remotes,
	}
}

func (s *RemoteService) Remotes() []*Remote {
	return s.remotes
}

func (s *RemoteService) FindByName(name string) *Remote {
	for _, r := range s.remotes {
		if r.Name() == name {
			return r
		}
	}
	return nil
}

func (s *RemoteService) FindByHost(host string) *Remote {
	for _, r := range s.remotes {
		if r.Host() == host {
			return r
		}
	}
	return nil
}
