package web

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/heyuuu/cube/project"
	"github.com/heyuuu/cube/util/slicekit"
)

// --- dto ---

type ProjectDTO struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	RepoUrl string   `json:"repoUrl"`
	Tags    []string `json:"tags"`
}

func toProjectDTO(entity *project.Project) *ProjectDTO {
	if entity == nil {
		return nil
	}

	return &ProjectDTO{
		Name:    entity.Name(),
		Path:    entity.Path(),
		RepoUrl: entity.RepoUrl(),
		Tags:    entity.Tags(),
	}
}

type WorkspaceDTO struct {
	Name       string   `json:"name"`
	Root       string   `json:"root"`
	PreferApps []string `json:"preferApps"`
	Scanner    any      `json:"scanner"`
}

func toWorkspaceDTO(entity *project.Workspace) *WorkspaceDTO {
	if entity == nil {
		return nil
	}

	return &WorkspaceDTO{
		Name:       entity.Name(),
		Root:       entity.Path(),
		PreferApps: entity.PreferApps(),
		Scanner:    toScannerResponseData(entity.Scanner()),
	}
}

func toScannerResponseData(scanner project.ProjectScanner) map[string]any {
	switch sc := scanner.(type) {
	case *project.GitProjectScanner:
		return map[string]any{
			"type":     "git",
			"maxDepth": sc.MaxDepth(),
		}
	default:
		return nil
	}
}

type RemoteDTO struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	DefaultPath string `json:"defaultPath"`
}

func toRemoteDTO(entity *project.Remote) *RemoteDTO {
	if entity == nil {
		return nil
	}

	return &RemoteDTO{
		Name:        entity.Name(),
		Host:        entity.Host(),
		DefaultPath: entity.DefaultPath(),
	}
}

// --- handler ---

type ProjectHandler struct {
	service *project.Service
}

func NewProjectHandler(service *project.Service) *ProjectHandler {
	return &ProjectHandler{
		service: service,
	}
}

func (h *ProjectHandler) Register(api huma.API) {
	apiGet(api, "/api/project/list", "", h.projectList)
	apiGet(api, "/api/project/info", "", h.projectInfo)

	apiGet(api, "/api/workspace/list", "", h.workspaceList)
	apiGet(api, "/api/workspace/info", "", h.workspaceInfo)

	apiGet(api, "/api/remote/list", "", h.remoteList)
	apiGet(api, "/api/remote/info", "", h.remoteInfo)
}

func (h *ProjectHandler) projectList(_ struct{}) (ListResult[*ProjectDTO], error) {
	projects := h.service.Projects()
	list := slicekit.Map(projects, toProjectDTO)
	return listResult(list), nil
}

func (h *ProjectHandler) projectInfo(input struct {
	Name string `json:"name"`
}) (result *ProjectDTO, err error) {
	proj := h.service.FindByName(input.Name)
	return toProjectDTO(proj), nil
}

func (h *ProjectHandler) workspaceList(_ struct{}) (ListResult[*WorkspaceDTO], error) {
	workspaces := h.service.Workspaces()
	list := slicekit.Map(workspaces, toWorkspaceDTO)
	return listResult(list), nil
}

func (h *ProjectHandler) workspaceInfo(input struct {
	Name string `json:"name"`
}) (result *WorkspaceDTO, err error) {
	ws := h.service.FindWorkspaceByName(input.Name)
	return toWorkspaceDTO(ws), nil
}

func (h *ProjectHandler) remoteList(_ struct{}) (ListResult[*RemoteDTO], error) {
	remotes := h.service.Remotes()
	list := slicekit.Map(remotes, toRemoteDTO)
	return listResult(list), nil
}

func (h *ProjectHandler) remoteInfo(input struct {
	Name string `json:"name"`
}) (result *RemoteDTO, err error) {
	remote := h.service.FindRemoteByName(input.Name)
	return toRemoteDTO(remote), nil
}
