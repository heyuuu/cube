package web

import (
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/project"
)

type ApiResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewApiResponse(ok bool, message string, data any) *ApiResponse {
	return &ApiResponse{ok, message, data}
}

type ProjectDTO struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	RepoUrl string   `json:"repoUrl"`
	Tags    []string `json:"tags"`
}

func ToProjectDTO(entity *project.Project) ProjectDTO {
	return ProjectDTO{
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

func ToWorkspaceDTO(entity *project.Workspace) WorkspaceDTO {
	return WorkspaceDTO{
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

type OpenerDTO struct {
	Name string `json:"name"`
	Bin  string `json:"bin"`
}

func ToOpenerDTO(entity *opener.Opener) OpenerDTO {
	return OpenerDTO{
		Name: entity.Name(),
		Bin:  entity.Bin(),
	}
}

type RemoteDTO struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	DefaultPath string `json:"defaultPath"`
}

func ToRemoteDTO(entity *project.Remote) RemoteDTO {
	return RemoteDTO{
		Name:        entity.Name(),
		Host:        entity.Host(),
		DefaultPath: entity.DefaultPath(),
	}
}
