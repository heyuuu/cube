//go:build wireinject

package app

import (
	"github.com/google/wire"

	"github.com/heyuuu/cube/config"
	handlers2 "github.com/heyuuu/cube/handlers"
	"github.com/heyuuu/cube/server"
	services2 "github.com/heyuuu/cube/services"
)

func InitApp() *App {
	wire.Build(
		// config
		config.Default,

		// server
		server.NewServer,

		// handlers
		handlers2.NewConfigHandler,
		handlers2.NewWorkspaceHandler,
		handlers2.NewProjectHandler,
		handlers2.NewApplicationHandler,
		handlers2.NewRemoteHandler,
		handlers2.AllHandlers,

		// services
		services2.NewConfigService,
		services2.NewWorkspaceService,
		services2.NewProjectService,
		services2.NewApplicationService,
		services2.NewRemoteService,
		services2.NewHistoryService,

		// app
		wire.Struct(new(App), "*"),
	)
	return nil
}
