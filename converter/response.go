package converter

import (
	"github.com/heyuuu/cube/dto/response"
	entities2 "github.com/heyuuu/cube/entities"
)

func ToProjectResponseDto(entity *entities2.Project) response.ProjectDto {
	return response.ProjectDto{
		Name:    entity.Name(),
		Path:    entity.Path(),
		RepoUrl: entity.RepoUrl(),
		Tags:    entity.Tags(),
	}
}

func ToWorkspaceResponseDto(entity *entities2.Workspace) response.WorkspaceDto {
	return response.WorkspaceDto{
		Name:       entity.Name(),
		Root:       entity.Path(),
		PreferApps: entity.PreferApps(),
		Scanner:    toScannerResponseData(entity.Scanner()),
	}
}

func toScannerResponseData(scanner entities2.ProjectScanner) map[string]any {
	switch sc := scanner.(type) {
	case *entities2.GitProjectScanner:
		return map[string]any{
			"type":     "git",
			"maxDepth": sc.MaxDepth(),
		}
	default:
		return nil
	}
}

func ToApplicationResponseDto(entity *entities2.Application) response.ApplicationDto {
	return response.ApplicationDto{
		Name: entity.Name(),
		Bin:  entity.Bin(),
	}
}

func ToRemoteResponseDto(entity *entities2.Remote) response.RemoteDto {
	return response.RemoteDto{
		Name:        entity.Name(),
		Host:        entity.Host(),
		DefaultPath: entity.DefaultPath(),
	}
}
