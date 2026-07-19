package cmd

import (
	"log/slog"
	"os"

	"github.com/heyuuu/cube/cmd/alfred"
	"github.com/heyuuu/cube/cmd/application"
	"github.com/heyuuu/cube/cmd/project"
	"github.com/heyuuu/cube/cmd/remote"
	"github.com/heyuuu/cube/cmd/workspace"
	config2 "github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/cube"
	"github.com/heyuuu/cube/logger"
	"github.com/heyuuu/cube/util/easycobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &easycobra.Command{
	Use:   "go-cube",
	Short: "go-cube " + cube.Version,
}

// 在 Execute 前执行全局 flags 的解析和应用
// notice: 不可使用 PersistentPreRun 或 PersistentPreRunE 替代，因为在没有定义 Run 相关操作的 Command 上不会调用 PersistentPreRun.
func rootPreExecute() error {
	cmd := rootCmd.CobraCommand()
	args := os.Args[1:]

	// persistent flags
	var cfgPath string
	var debug bool
	cmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "config folder path (default is ~/.go-cube/)")
	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "open debug mode")
	cmd.PersistentFlags().ParseErrorsAllowlist.UnknownFlags = true
	err := cmd.PersistentFlags().Parse(args)
	if err != nil {
		return err
	}

	// 设置 debug 环境
	config2.SetDebug(debug)

	// 初始化配置
	config2.InitConfig(cfgPath)

	// 初始化 Logger
	logger.Init()

	// 记录启动日志
	slog.Debug("command start", "debug", debug, "cfgPath", config2.ConfigPath(), "args", args)

	return nil
}

func init() {
	rootCmd.AddCommand(
		// group commands
		project.ProjectCmd,
		application.AppCmd,
		remote.RemoteCmd,
		workspace.WorkspaceCmd,
		alfred.AlfredCmd,
		// simple commands
		versionCmd,
		configCmd,
		serverCmd,
	)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootPreExecute()
	if err != nil {
		slog.Error("pre execute failed", "err", err)
		os.Exit(1)
	}

	err = rootCmd.Execute()
	if err != nil {
		slog.Error("execute failed", "err", err)
		os.Exit(1)
	}
}
