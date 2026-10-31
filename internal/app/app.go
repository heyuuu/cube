package app

import (
	"errors"
	"fmt"
	"slices"

	"github.com/heyuuu/x/di"

	"cube/internal/config"
	"cube/internal/logger"
	"cube/internal/server"
)

// App 进程级上下文：New() 只建空壳供 cmd 构建期持有，Init() 完成组装，
// Init 之前调用读 API / 生命周期方法会 panic。
// 内部只持有冻结后的 di 容器，其余成员经 di.Get 按需解析。
type App struct {
	hasInit   bool
	cfgFile   string // 配置文件路径（--config 或默认值）
	debug     bool   // --debug：只影响 logger 初始化
	local     bool   // --local：query 缺省的命令以 cwd 定位项目（cmd/helpers.go pickProject 消费）
	container *di.Container
}

// New 返回未初始化的空壳 App，其余方法须待 Init 成功后方可调用。
func New() *App {
	return &App{}
}

// Init 加载配置并完成组装；成功后不可再次调用。
func (a *App) Init(cfgFile string, debug bool, local bool) error {
	if a.hasInit {
		return errors.New("app 已初始化过，不可重复初始化")
	}

	// 初始化配置
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	paths := config.MakePaths(cfg.DataDir)

	// 尽量在其他行为前初始化 Logger
	logger.Init(paths.LogDir(), debug)

	// 初始化 DI 容器
	// 业务单例都在此注册，方便自动注入
	c, err := di.RegisterAll(
		slices.Concat(
			/* 默认对象 */
			[]any{
				di.Value(cfg),
				di.Value(cfg.Server),
				di.Value(paths),
				server.NewServer,
			},
			/* services */
			bootFactories(),
		)...,
	)
	if err != nil {
		return err
	}

	// 设置参数
	a.cfgFile = cfgFile
	a.debug = debug
	a.local = local
	a.container = c
	a.hasInit = true
	return nil
}

func (a *App) checkInit() {
	if !a.hasInit {
		panic("app 必须先 Init() 后使用，请检查代码逻辑")
	}
}

// Local 返回 --local 标志是否生效（以 cwd 定位项目）。
func (a *App) Local() bool {
	a.checkInit()
	return a.local
}

// Container 返回冻结后的 di 容器（Init 后可用）。
func (a *App) Container() *di.Container {
	a.checkInit()
	return a.container
}

// StartBackgroundJobs 启动常驻进程的后台任务（分发到各 service 的 OnServerStart 钩子）。
// 仅常驻 server 调用；CLI 短命进程不调用。
func (a *App) StartBackgroundJobs() {
	for s := range di.AllOf[serverStartHook](a.container) {
		s.OnServerStart()
	}
}

// StopBackgroundJobs 停止后台任务（server 退出时调，分发到 OnServerStop 钩子）。
func (a *App) StopBackgroundJobs() {
	for s := range di.AllOf[serverStopHook](a.container) {
		s.OnServerStop()
	}
}

/* 快捷方法 */

// mustGet 从 DI 容器里获取实例
func mustGet[T any](a *App) T {
	a.checkInit()
	return di.MustGet[T](a.container)
}

// Server 返回 HTTP 服务（Init 后可用）。
func (a *App) Server() *server.Server { return mustGet[*server.Server](a) }
