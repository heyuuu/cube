// Package env 承载 CLI 命令的运行环境：持有全局 flag 值，按需执行
// config→logger→app 装配链；命令工厂收 *Env，RunE 里调 env.App() 取依赖。
//
// 独立子包（不放 cmd 包内）是因为 cmd/alfred、cmd/server 等子命令组的
// 工厂也要收 *Env，而它们不能反向 import cmd（循环依赖）。
//
// New 不带参数：Env 必须先于命令树创建，此时 flag 尚未解析；
// flag 值由 root 的 PersistentPreRunE 在解析完成后经 Init 注入。
package env

import (
	"errors"
	"fmt"

	"cube/app"
)

type Env struct {
	hasInit bool
	cfgFile string // 配置文件路径（--config 或默认值）
	debug   bool   // --debug：只影响 logger 初始化
	local   bool   // --local：query 缺省的命令以 cwd 定位项目（cmd/helpers.go pickProject 消费）
	app     *app.App
}

func New() *Env {
	return &Env{}
}

// Init 注入全局 flag 值并执行装配链。重复调用直接报错——每条命令路径只应
// 有一个触发点（root 的 PersistentPreRunE），重复即接线错误，宁可炸出来。
func (e *Env) Init(cfgFile string, debug bool, local bool) error {
	if e.hasInit {
		return errors.New("env 已初始化过，不可重复初始化")
	}
	
	// 初始化 App
	a, err := app.Init(cfgFile, debug)
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

// App 返回装配完成的 App；未 Init 即调用属编程错误，panic 快速暴露。
func (e *Env) App() *app.App {
	e.checkInit()
	return e.app
}
