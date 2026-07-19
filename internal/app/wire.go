//go:build wireinject

package app

import (
	"github.com/google/wire"
	"github.com/heyuuu/cube/internal/config"
	"github.com/heyuuu/cube/internal/handlers"
	"github.com/heyuuu/cube/internal/server"
	"github.com/heyuuu/cube/internal/services"
)

func InitApp() *App {
	wire.Build(
		// config
		config.Default,

		// server
		server.NewServer,

		// handlers
		handlers.NewConfigHandler,
		handlers.NewWorkspaceHandler,
		handlers.NewProjectHandler,
		handlers.NewApplicationHandler,
		handlers.NewRemoteHandler,
		handlers.AllHandlers,

		// services
		services.NewConfigService,
		services.NewWorkspaceService,
		services.NewProjectService,
		services.NewApplicationService,
		services.NewRemoteService,
		services.NewHistoryService,

		// app
		wire.Struct(new(App), "*"),
	)
	return nil
}
