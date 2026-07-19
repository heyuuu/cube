package app

import (
	"github.com/heyuuu/cube/server"
	services2 "github.com/heyuuu/cube/services"
)

type App struct {
	server *server.Server

	workspaceService   *services2.WorkspaceService
	projectService     *services2.ProjectService
	applicationService *services2.ApplicationService
	remoteService      *services2.RemoteService
	historyService     *services2.HistoryService
}

func (app *App) Server() *server.Server {
	return app.server
}

func (app *App) WorkspaceService() *services2.WorkspaceService {
	return app.workspaceService
}

func (app *App) ProjectService() *services2.ProjectService {
	return app.projectService
}

func (app *App) ApplicationService() *services2.ApplicationService {
	return app.applicationService
}

func (app *App) RemoteService() *services2.RemoteService {
	return app.remoteService
}

func (app *App) HistoryService() *services2.HistoryService {
	return app.historyService
}
