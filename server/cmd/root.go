package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"cube/cmd/alfred"
	"cube/cmd/dev"
	"cube/cmd/env"
	"cube/cmd/server"
	"cube/cmd/ui"
	"cube/version"
)

func Execute() {
	// 构建 cmd
	env := env.New()
	cmd := newRootCmd(env)

	// 执行命令
	if err := cmd.Execute(); err != nil {
		slog.Error("命令执行失败", "err", err)
		os.Exit(1)
	}
}

// 默认配置文件路径，区分开发环境、正式环境
func defaultConfigPath() string {
	if version.IsDev() {
		return "~/.config/" + version.AppName + "-dev/config.json"
	}
	return "~/.config/" + version.AppName + "/config.json"
}

func newRootCmd(env *env.Env) *cobra.Command {
	var cfgFile string
	var debug bool
	var local bool

	cmd := &cobra.Command{
		Use:   version.AppName,
		Short: version.AppName + " " + version.Version(),
		Long: `cube —— 面向个人开发者的本地多项目管理工具（CLI 优先 + 本地 Web）。

命令按领域分组：
  - 项目：list(列表) info(详情) open(打开) init/clone(初始化) check(检查)
  - opener：openers(列表) open-path(打开路径) diff(对比)
  - git：push(批量推送) pull(批量拉取)
  - Web：server(本地服务) ui(打开 Web UI) openapi(导出 API spec)

配置默认在 ~/.config/cube/，全局 flag -c 可覆盖配置目录，-d 开 debug 日志。
--local 让 query 缺省的命令（info/pull/push/open）以当前目录定位项目，
等同在命令末尾补 query 为 "."。`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true // 钩子之前发生的输入类错误仍附 usage；运行期错误不再附 usage
			return env.Init(cfgFile, debug, local)
		},
	}

	// cmd 上绑定全局 flag，仅用于生成 help 提示(此时 --config/--debug/--local 早解析完了)
	cmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfigPath(), fmt.Sprintf("config file (default is %s)", defaultConfigPath()))
	cmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false, "enable debug mode")
	cmd.PersistentFlags().BoolVar(&local, "local", false, "query 缺省时以当前目录定位项目（shell 函数 p 即此模式）")

	// 绑定子命令
	registerSubCommands(cmd, env)

	return cmd
}

func registerSubCommands(cmd *cobra.Command, env *env.Env) {
	cmd.AddCommand(newVersionCmd(env))

	// web server 相关
	cmd.AddCommand(server.NewCmd(env))
	cmd.AddCommand(ui.NewCmd(env))
	cmd.AddCommand(newOpenapiCmd(env))

	// project 相关
	cmd.AddCommand(newListCmd(env))      // 项目列表
	cmd.AddCommand(newInfoCmd(env))      // 项目信息
	cmd.AddCommand(newOpenCmd(env))      // 打开项目
	cmd.AddCommand(newPathCmd(env))      // 输出项目路径（供 shell 包装函数 cd）
	cmd.AddCommand(newInitCmd(env))      // 初始化空项目
	cmd.AddCommand(newCreateCmd(env))    // 使用模板初始化项目
	cmd.AddCommand(newCloneCmd(env))     // 使用 RepoUrl 初始化项目
	cmd.AddCommand(newWorkspaceCmd(env)) // monorepo workspace 声明管理

	// open 相关
	cmd.AddCommand(newOpenersCmd(env))
	cmd.AddCommand(newOpenPathCmd(env))
	cmd.AddCommand(newDiffCmd(env))

	// forge（git 托管平台配置，1040）
	cmd.AddCommand(newForgeCmd(env))

	// git 相关
	cmd.AddCommand(newPushCmd(env))
	cmd.AddCommand(newPullCmd(env))

	// 内部命令
	cmd.AddCommand(alfred.NewCmd(env))
	cmd.AddCommand(dev.NewCmd(env))

	cmd.AddCommand(newDoctorCmd(env)) // 环境体检

	// 待整理命令
	cmd.AddCommand(newCheckCmd(env))
}
