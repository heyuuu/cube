package env

import (
	"errors"
	"fmt"

	"cube/app"
	"cube/config"
	"cube/logger"
)

type Env struct {
	hasInit bool
	cfgFile string
	debug   bool
	local   bool
	app     *app.App
}

func New() *Env {
	return &Env{}
}

func (e *Env) Init(cfgFile string, debug bool, local bool) error {
	if e.hasInit {
		return errors.New("env 已初始化过，不可重复初始化")
	}

	// 初始化配置
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	// 尽量在其他行为前初始化 Logger
	logger.Init(cfg.Log, debug)

	// 初始化 App
	a, err := app.New(cfg)
	if err != nil {
		return fmt.Errorf("app 初始化失败: %w", err)
	}

	// 设置参数
	e.cfgFile = cfgFile
	e.debug = debug
	e.local = local
	e.app = a
	e.hasInit = true
	return nil
}

func (e *Env) checkInit() {
	if !e.hasInit {
		panic("env 必须先 Init() 后使用，请检查代码逻辑")
	}
}

func (e *Env) CfgFile() string {
	e.checkInit()
	return e.cfgFile
}

func (e *Env) Debug() bool {
	e.checkInit()
	return e.debug
}

func (e *Env) Local() bool {
	e.checkInit()
	return e.local
}

func (e *Env) App() *app.App {
	e.checkInit()
	return e.app
}
