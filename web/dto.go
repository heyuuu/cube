package web

import (
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/project"
)

// API 响应的基础结构
type ApiResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewApiResponse(ok bool, message string, data any) *ApiResponse {
	return &ApiResponse{ok, message, data}
}

type ProjectDto struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	RepoUrl string   `json:"repoUrl"`
	Tags    []string `json:"tags"`
}

func ToProjectResponseDto(entity *project.Project) ProjectDto {
	return ProjectDto{
		Name:    entity.Name(),
		Path:    entity.Path(),
		RepoUrl: entity.RepoUrl(),
		Tags:    entity.Tags(),
	}
}

type WorkspaceDto struct {
	Name       string   `json:"name"`
	Root       string   `json:"root"`
	PreferApps []string `json:"preferApps"`
	Scanner    any      `json:"scanner"`
}

func ToWorkspaceResponseDto(entity *project.Workspace) WorkspaceDto {
	return WorkspaceDto{
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

type ApplicationDto struct {
	Name string `json:"name"`
	Bin  string `json:"bin"`
}

func ToApplicationResponseDto(entity *opener.Application) ApplicationDto {
	return ApplicationDto{
		Name: entity.Name(),
		Bin:  entity.Bin(),
	}
}

type RemoteDto struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	DefaultPath string `json:"defaultPath"`
}

func ToRemoteResponseDto(entity *project.Remote) RemoteDto {
	return RemoteDto{
		Name:        entity.Name(),
		Host:        entity.Host(),
		DefaultPath: entity.DefaultPath(),
	}
}
