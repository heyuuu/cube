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
	apiGet(api, "/api/project/scan-rules", "", h.scanRules)
	apiGet(api, "/api/project/clone-rules", "", h.cloneRules)
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

func (h *ProjectHandler) scanRules(_ struct{}) (ListResult[project.ScanRule], error) {
	rules := h.service.ScanRules()
	return listResult(rules), nil
}

func (h *ProjectHandler) cloneRules(_ struct{}) (ListResult[project.CloneRule], error) {
	rules := h.service.CloneRules()
	return listResult(rules), nil
}
