package cmd

import (
	"log/slog"
	"os"

	"github.com/heyuuu/cube/cmd/alfred"
	"github.com/heyuuu/cube/cmd/opener"
	"github.com/heyuuu/cube/cmd/project"
	"github.com/heyuuu/cube/cmd/util/easycobra"
	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/db"
	"github.com/heyuuu/cube/history"
	"github.com/heyuuu/cube/logger"
	"github.com/heyuuu/cube/version"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &easycobra.Command{
	Use:   "go-cube",
	Short: "go-cube " + version.Version,
	Children: []*easycobra.Command{
		// group commands
		project.RootCmd,
		opener.RootCmd,
		alfred.RootCmd,
		// simple commands
		versionCmd,
		configCmd,
		serverCmd,
	},
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
	config.SetDebug(debug)

	// 初始化配置
	err = config.InitConfig(cfgPath)
	if err != nil {
		return err
	}

	// 初始化数据文件 data.db
	err = db.Init(config.ConfigPath(),
		&history.ProjectSelectLog{},
		&history.ProjectOpenLog{},
	)
	if err != nil {
		return err
	}

	// 初始化 Logger
	logger.Init()

	// 记录启动日志
	slog.Debug("command start", "debug", debug, "cfgPath", config.ConfigPath(), "args", args)

	return nil
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
