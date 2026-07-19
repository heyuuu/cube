package web

import (
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/project"
)

func ToProjectResponseDto(entity *project.Project) ProjectDto {
	return ProjectDto{
		Name:    entity.Name(),
		Path:    entity.Path(),
		RepoUrl: entity.RepoUrl(),
		Tags:    entity.Tags(),
	}
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

func ToApplicationResponseDto(entity *opener.Application) ApplicationDto {
	return ApplicationDto{
		Name: entity.Name(),
		Bin:  entity.Bin(),
	}
}

func ToRemoteResponseDto(entity *project.Remote) RemoteDto {
	return RemoteDto{
		Name:        entity.Name(),
		Host:        entity.Host(),
		DefaultPath: entity.DefaultPath(),
	}
}
