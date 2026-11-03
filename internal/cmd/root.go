package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/alfred"
	"cube/internal/cmd/cmdutil"
	"cube/internal/cmd/dev"
	"cube/internal/cmd/project"
	"cube/internal/cmd/server"
	"cube/internal/cmd/tpl"
	"cube/internal/version"
)

// Execute 构建并执行根命令，失败时打日志并以非零码退出。
func Execute() {
	// 构建 cmd
	cmd := newRootCmd(app.New())

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

func newRootCmd(a *app.App) *cobra.Command {
	var cfgFile string
	var debug bool
	var local bool

	cmd := &cobra.Command{
		Use:   version.AppName,
		Short: version.AppName + " " + version.Version(),
		Long: `cube —— 面向个人开发者的本地多项目管理工具（Web UI 优先，CLI 简单可行兜底）。

命令按领域分组：
  - 项目：list(列表) info(详情) open(打开) init/clone(初始化) check(检查)
  - opener：openers(列表) open-path(打开路径) diff(对比)
  - git：push(批量推送) pull(批量拉取)
  - Web：server(本地服务) ui(打开 Web UI) openapi(导出 API spec)

配置默认在 ~/.config/cube/（dev 为 cube-dev），全局 flag --config 可覆盖配置文件路径，--debug 开 debug 日志。
--local 让 query 缺省的命令（info/pull/push/open）以当前目录定位项目，
等同在命令末尾补 query 为 "."。`,
		// 惰性装配：命令真正执行前才 Init（config→logger→app），
		// --help / 未知命令等不触发 RunE 的路径全程零装配。
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// PersistentPreRunE 之前发生的输入类错误仍附 usage；运行期错误不再附 usage
			cmd.SilenceUsage = true
			// 展开 cfgFile 路径
			absCfgFile, err := cmdutil.ExtendPath(cfgFile)
			if err != nil {
				return fmt.Errorf("解析 config 路径失败: %w", err)
			}
			return a.Init(absCfgFile, debug, local)
		},
	}

	// 绑定全局 flag
	cmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfigPath(), "config file")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")
	cmd.PersistentFlags().BoolVar(&local, "local", false, "query 缺省时以当前目录定位项目（shell 函数 p 即此模式）")

	// 绑定子命令
	registerSubCommands(cmd, a)

	return cmd
}

func registerSubCommands(cmd *cobra.Command, a *app.App) {
	cmd.AddCommand(newVersionCmd())

	// web server 相关
	cmd.AddCommand(server.NewCmd(a))
	cmd.AddCommand(newUICmd(a))
	cmd.AddCommand(newOpenapiCmd(a))

	// project 相关
	cmd.AddCommand(project.NewCmd(a))  // project 相关
	cmd.AddCommand(newOpenCmd(a))      // 打开项目
	cmd.AddCommand(newPathCmd(a))      // 输出项目路径（供 shell 包装函数 cd）
	cmd.AddCommand(newCloneCmd(a))     // 使用 RepoURL 初始化项目
	cmd.AddCommand(newWorkspaceCmd(a)) // monorepo workspace 声明管理

	// tpl 相关
	cmd.AddCommand(tpl.NewCmd(a))

	// open 相关
	cmd.AddCommand(newOpenersCmd(a))
	cmd.AddCommand(newOpenPathCmd(a))
	cmd.AddCommand(newDiffCmd(a))

	// forge（git 托管平台配置，1040）
	cmd.AddCommand(newForgeCmd(a))

	// git 相关
	cmd.AddCommand(newPushCmd(a))
	cmd.AddCommand(newPullCmd(a))

	// 内部命令
	cmd.AddCommand(alfred.NewCmd(a))
	cmd.AddCommand(dev.NewCmd(a))

	cmd.AddCommand(newDoctorCmd(a)) // 环境体检

	// 待整理命令
	cmd.AddCommand(newCheckCmd(a))

	cmd.AddCommand(server.NewCmd(a)) // server 相关

	/* <<subcommand inject point>> */

	// 绑定命令别名
	registerAliasCommands(cmd, map[string][]string{
		// project
		"list": {"project", "list"},
		"info": {"project", "info"},
		// tpl
		"create": {"tpl", "create"},
	})
}

func registerAliasCommands(cmd *cobra.Command, aliases map[string][]string) {
	for name, alias := range aliases {
		registerAliasCommand(cmd, name, alias)
	}
}

func registerAliasCommand(cmd *cobra.Command, name string, alias []string) {
	target := findCommand(cmd, alias)
	if target == nil {
		// 注册期错误属于代码 bug，启动即暴露
		panic(fmt.Sprintf("别名 %s 的目标命令 %v 不存在", name, alias))
	}
	// 浅拷贝：flag set 与 RunE 闭包共享，别名命令与目标命令行为始终一致
	clone := *target
	clone.Use = name
	clone.Aliases = nil
	clone.Short = fmt.Sprintf("命令 %s 的别名", strings.Join(alias, " "))
	clone.Long = ""
	cmd.AddCommand(&clone)
}

// findCommand 按路径逐层查找子命令，任一层未命中返回 nil。
func findCommand(root *cobra.Command, path []string) *cobra.Command {
	cur := root
	for _, name := range path {
		next, _, err := cur.Find([]string{name})
		if err != nil || next == cur {
			return nil
		}
		cur = next
	}
	return cur
}
