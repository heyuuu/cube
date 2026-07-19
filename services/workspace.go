package services

import (
	"strings"

	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/entities"
	"github.com/heyuuu/cube/util/slicekit"
)

type WorkspaceService struct {
	workspaces []*entities.Workspace
}

func NewWorkspaceService(conf config.Config) *WorkspaceService {
	workspaces := slicekit.Map(conf.Workspaces, entities.NewWorkspace)

	return &WorkspaceService{
		workspaces: workspaces,
	}
}

func (s *WorkspaceService) Workspaces() []*entities.Workspace {
	return s.workspaces
}

func (s *WorkspaceService) FindByName(name string) *entities.Workspace {
	for _, ws := range s.workspaces {
		if ws.Name() == name {
			return ws
		}
	}
	return nil
}

func (s *WorkspaceService) FindByProjectName(projectName string) *entities.Workspace {
	if wsName, _, ok := strings.Cut(projectName, ":"); ok {
		return s.FindByName(wsName)
	}
	return nil
}
