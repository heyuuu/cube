package app

import (
	"github.com/heyuuu/cube/history"
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/project"
	"github.com/heyuuu/cube/web"
)

type App struct {
	server *web.Server

	workspaceService   *project.WorkspaceService
	projectService     *project.ProjectService
	applicationService *opener.ApplicationService
	remoteService      *project.RemoteService
	historyService     *history.HistoryService
}

func (app *App) Server() *web.Server {
	return app.server
}

func (app *App) WorkspaceService() *project.WorkspaceService {
	return app.workspaceService
}

func (app *App) ProjectService() *project.ProjectService {
	return app.projectService
}

func (app *App) ApplicationService() *opener.ApplicationService {
	return app.applicationService
}

func (app *App) RemoteService() *project.RemoteService {
	return app.remoteService
}

func (app *App) HistoryService() *history.HistoryService {
	return app.historyService
}
