//go:build wireinject

package app

import (
	"github.com/google/wire"

	"github.com/heyuuu/cube/config"
	services2 "github.com/heyuuu/cube/history"
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/project"
	"github.com/heyuuu/cube/web"
)

func InitApp() *App {
	wire.Build(
		// config
		config.Default,

		// server
		web.NewServer,

		// handlers
		web.NewConfigHandler,
		web.NewWorkspaceHandler,
		web.NewProjectHandler,
		web.NewApplicationHandler,
		web.NewRemoteHandler,
		web.AllHandlers,

		// services
		config.NewConfigService,
		project.NewWorkspaceService,
		project.NewProjectService,
		opener.NewApplicationService,
		project.NewRemoteService,
		services2.NewHistoryService,

		// app
		wire.Struct(new(App), "*"),
	)
	return nil
}
