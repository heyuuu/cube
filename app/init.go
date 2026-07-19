package app

import (
	"sync"

	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/db"
	"github.com/heyuuu/cube/history"
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/project"
	"github.com/heyuuu/cube/web"
)

var (
	defaultApp  *App
	defaultOnce sync.Once
)

func Default() *App {
	defaultOnce.Do(func() {
		defaultApp = InitApp()
	})
	return defaultApp
}

func InitApp() *App {
	configConfig := config.Default()
	defaultDB := db.Default()

	configService := config.NewConfigService(configConfig)

	configHandler := web.NewConfigHandler(configService)
	workspaceService := project.NewWorkspaceService(configConfig)
	workspaceHandler := web.NewWorkspaceHandler(workspaceService)

	projectService := project.NewProjectService(workspaceService)
	projectHandler := web.NewProjectHandler(projectService)

	applicationService := opener.NewApplicationService(configConfig)
	applicationHandler := web.NewApplicationHandler(applicationService)

	remoteService := project.NewRemoteService(configConfig)
	remoteHandler := web.NewRemoteHandler(remoteService)

	v := web.AllHandlers(configHandler, workspaceHandler, projectHandler, applicationHandler, remoteHandler)
	server := web.NewServer(v)

	historyService := history.NewHistoryService(defaultDB)

	return &App{
		server:             server,
		workspaceService:   workspaceService,
		projectService:     projectService,
		applicationService: applicationService,
		remoteService:      remoteService,
		historyService:     historyService,
	}
}
