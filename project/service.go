package project

import (
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/util/matcher"
	"github.com/heyuuu/cube/util/slicekit"
)

type Service struct {
	workspaces []*Workspace
	remotes    []*Remote
	scanCache  map[string][]*Project
	lockPool   sync.Map
}

func NewProjectService(conf config.Config) *Service {
	workspaces := slicekit.Map(conf.Workspaces, NewWorkspace)
	remotes := slicekit.Map(conf.Remotes, NewRemote)

	return &Service{
		workspaces: workspaces,
		remotes:    remotes,
		scanCache:  make(map[string][]*Project),
	}
}

// --- workspace --

func (s *Service) Workspaces() []*Workspace {
	return s.workspaces
}

func (s *Service) FindWorkspaceByName(name string) *Workspace {
	for _, ws := range s.workspaces {
		if ws.Name() == name {
			return ws
		}
	}
	return nil
}

func (s *Service) FindWorkspaceByProjectName(projectName string) *Workspace {
	if wsName, _, ok := strings.Cut(projectName, ":"); ok {
		return s.FindWorkspaceByName(wsName)
	}
	return nil
}

// -- remote --

func (s *Service) Remotes() []*Remote {
	return s.remotes
}

func (s *Service) FindRemoteByName(name string) *Remote {
	for _, r := range s.remotes {
		if r.Name() == name {
			return r
		}
	}
	return nil
}

func (s *Service) FindRemoteByHost(host string) *Remote {
	for _, r := range s.remotes {
		if r.Host() == host {
			return r
		}
	}
	return nil
}

// --- project --

func (s *Service) Projects() []*Project {
	workspaces := s.Workspaces()
	projectsGroup := slicekit.Map(workspaces, func(ws *Workspace) []*Project {
		return s.ScanProjects(ws)
	})
	return slices.Concat(projectsGroup...)
}

func (s *Service) FindByName(name string) *Project {
	ws := s.FindWorkspaceByProjectName(name)
	if ws == nil {
		return nil
	}

	for _, project := range s.ScanProjects(ws) {
		if project.Name() == name {
			return project
		}
	}
	return nil
}

func (s *Service) Search(query string) []*Project {
	return s.SearchInWorkspace(query, "")
}

func (s *Service) SearchInWorkspace(query string, workspaceName string) []*Project {
	projects := s.projectsInWorkspace(workspaceName)
	if len(projects) == 0 {
		return nil
	}

	if len(query) == 0 {
		return projects
	}

	projectMatcher := matcher.NewKeywordMatcher(projects, (*Project).Name, nil)
	return projectMatcher.Match(query)
}

func (s *Service) projectsInWorkspace(workspaceName string) []*Project {
	if workspaceName == "" {
		return s.Projects()
	} else {
		ws := s.FindWorkspaceByName(workspaceName)
		if ws == nil {
			return nil
		}

		return s.ScanProjects(ws)
	}
}

func (s *Service) getLock(key string) *sync.RWMutex {
	lock, _ := s.lockPool.LoadOrStore(key, &sync.RWMutex{})
	return lock.(*sync.RWMutex)
}

func (s *Service) ScanProjects(ws *Workspace) []*Project {
	// 判断是否有扫描规则，若没有直接返回
	scanner := ws.Scanner()
	if scanner == nil {
		return nil
	}

	// 获取锁
	lock := s.getLock(ws.Name())

	// 先尝试读缓存
	lock.RLock()
	if projects, ok := s.scanCache[ws.Name()]; ok {
		lock.RUnlock()
		return projects
	}
	lock.RUnlock()

	// 缓存未命中，实际扫描本地文件
	lock.Lock()
	defer lock.Unlock()

	projects, err := scanner.Scan(ws)
	if err != nil {
		log.Print(err)
	}

	// 更新缓存(即使有 err 也更新，避免重复扫描)
	s.scanCache[ws.Name()] = projects
	return projects
}
