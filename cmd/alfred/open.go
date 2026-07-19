package alfred

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/cmd/util/easycobra"
)

// cmd `alfred project-open`
var projectOpenCmd = &easycobra.Command{
	Use:   "project-open {project : 项目名} {--app= : 打开项目的App}",
	Short: "打开项目。非交互模式只支持准确项目名，非交互模式下支持模糊搜索",
	Args:  cobra.ExactArgs(1),
	InitRun: func(cmd *cobra.Command) easycobra.Run {
		// init flags
		var appName string
		cmd.Flags().StringVar(&appName, "app", "", "打开项目的App")

		// run
		return func(args []string) error {
			projectName := args[0]

			// history: 记录打开项目的程序
			app.Default().HistoryService().AddProjectOpenLog(projectName, appName, true)

			appService := app.Default().OpenerService()
			projService := app.Default().ProjectService()

			// 匹配项目
			proj := projService.FindByName(projectName)
			if proj == nil {
				return errors.New("未找到指定项目: " + projectName)
			}

			// 获取打开项目的app
			openApp := appService.FindByName(appName)
			if openApp == nil {
				return errors.New("未找到指定app: " + appName)
			}

			// 打开项目
			err := passthruRun(openApp.Bin(), proj.Path())
			if err != nil {
				return fmt.Errorf("打开失败: %w", err)
			}

			return nil
		}
	},
}

func passthruRun(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
