package app

import (
	"fmt"
	"path/filepath"

	"cube/internal/config"
	"cube/internal/domain/forge"
	"cube/internal/domain/opener"
	"cube/internal/domain/project"
	"cube/internal/domain/template"
	"cube/internal/domain/usage"
	"cube/internal/domain/workbench"
	"cube/internal/handlers"
	"cube/internal/logger"
	server2 "cube/internal/server"
	"cube/web"
)

type App struct {
	cfg    *config.Config
	paths  *Paths
	server *server2.Server

	// services 有序清单，供生命周期钩子（OnServerStart / OnServerStop）分发
	services []any

	projectService   *project.Service
	workbenchService *workbench.Service
	openerService    *opener.Service
	usageService     *usage.Service
	templateService  *template.Service
	forgeService     *forge.Service
}

func Init(cfgFile string, debug bool) (*App, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}

	paths := NewPaths(cfg.DataDir)

	// 尽量在其他行为前初始化 Logger
	logger.Init(paths.LogDir(), debug)

	// 组装 services
	projectService := project.NewService(paths.SettingsFile(), paths.CacheDir())
	openerService := opener.NewService(paths.SettingsFile(), nil, server2.BaseURL(cfg.Server))
	usageService := usage.NewService(paths.StateDir())
	workbenchService := workbench.NewService(projectService.RefreshGitInfo)
	templateService := template.NewService(paths.SettingsFile())
	forgeService := forge.NewService(paths.SettingsFile(), filepath.Join(paths.CacheDir(), "forge-repos.json"))
	services := []any{projectService, openerService, usageService, workbenchService, templateService, forgeService}

	// 组装 web server
	configHandler := handlers.NewConfigHandler(cfg)
	projectHandler := handlers.NewProjectHandler(projectService, openerService, usageService)
	openerHandler := handlers.NewOpenerHandler(openerService)
	mdHandler := handlers.NewMdHandler()
	iconHandler := handlers.NewIconHandler()
	workbenchHandler := handlers.NewWorkbenchHandler(workbenchService)
	forgeHandler := handlers.NewForgeHandler(forgeService, projectService)
	templateHandler := handlers.NewTemplateHandler(templateService)
	usageHandler := handlers.NewUsageHandler(usageService)
	server := server2.NewServer(
		cfg.Server,
		[]server2.Handler{
			// 系统端点
			server2.NewSystemHandler(),
			// 静态资源端点
			server2.NewStaticHandler(web.FS()),
			// 业务端点
			configHandler,
			projectHandler,
			openerHandler,
			mdHandler,
			workbenchHandler,
			forgeHandler,
			templateHandler,
			usageHandler,
			iconHandler,
		},
	)

	return &App{
		cfg:    cfg,
		paths:  paths,
		server: server,

		services:         services,
		projectService:   projectService,
		workbenchService: workbenchService,
		openerService:    openerService,
		usageService:     usageService,
		templateService:  templateService,
		forgeService:     forgeService,
	}, nil
}

func (a *App) Config() *config.Config  { return a.cfg }
func (a *App) Paths() *Paths           { return a.paths }
func (a *App) Server() *server2.Server { return a.server }

func (a *App) ProjectService() *project.Service   { return a.projectService }
func (a *App) OpenerService() *opener.Service     { return a.openerService }
func (a *App) UsageService() *usage.Service       { return a.usageService }
func (a *App) TemplateService() *template.Service { return a.templateService }
func (a *App) ForgeService() *forge.Service       { return a.forgeService }

// StartBackgroundJobs 启动常驻进程的后台任务（分发到各 service 的 OnServerStart 钩子）。
// 仅常驻 server 调用；CLI 短命进程不调用。
func (a *App) StartBackgroundJobs() {
	for _, s := range a.services {
		if h, ok := s.(serverStartHook); ok {
			h.OnServerStart()
		}
	}
}

// StopBackgroundJobs 停止后台任务（server 退出时调，分发到 OnServerStop 钩子）。
func (a *App) StopBackgroundJobs() {
	for _, s := range a.services {
		if h, ok := s.(serverStopHook); ok {
			h.OnServerStop()
		}
	}
}
