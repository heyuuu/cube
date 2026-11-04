// Package ui 提供 `cube ui` 命令组，提供ui相关命令
package ui

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
	"cube/internal/server/client"
)

// NewCmd 是 `cube ui` 命令组入口，纯分发。
func NewCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "打开 Web UI（首页 / md 页 / workbench 页）",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWeb(a)
		},
	}

	cmd.AddCommand(newMdCmd(a))
	cmd.AddCommand(newWorkbenchCmd(a))
	/* <<subcommand inject point>> */

	return cmd
}

// runWeb 打开 Web UI 首页。
func runWeb(a *app.App) error {
	client := client.NewClient(a.Server().ServerURL())
	if st := client.Status(); !st.Running {
		return errors.New("server 未运行，请先执行: cube server start")
	}
	openInBrowser(a.Server().ServerURL())
	return nil
}

// openPage ui 命令族共用的「路径 → 页面」流程：路径解析（~ / 相对 / 裸文件名补 ./，
// cwd 解析属出口层职责）→ 存在性校验 → server 探活 → 拼 路由+QueryEscape(path) 开浏览器。
// route 形如 "md" / "workbench"（前端路由段）。
func openPage(a *app.App, route string, rawPath string) error {
	if rawPath[0] != '/' && rawPath[0] != '~' && rawPath[0] != '.' {
		rawPath = "./" + rawPath
	}
	absPath, err := cmdutil.ExtendPath(rawPath)
	if err != nil {
		return fmt.Errorf("解析路径失败: %w", err)
	}
	// 目录也放行：/md 页对目录展示左侧文件树（无 md 的目录显示空态）
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("路径不存在: %s", absPath)
	}

	client := client.NewClient(a.Server().ServerURL())
	if st := client.Status(); !st.Running {
		return errors.New("server 未运行，请先执行: cube server start")
	}

	openInBrowser(fmt.Sprintf("%s%s?path=%s", a.Server().ServerURL(), route, url.QueryEscape(absPath)))
	return nil
}

// openInBrowser 用系统 open 打开 URL。失败不阻断：打出 URL 让用户手动访问。
func openInBrowser(pageURL string) {
	if err := exec.Command("open", pageURL).Run(); err != nil {
		fmt.Printf("自动打开浏览器失败，请手动访问：%s\n", pageURL)
	} else {
		fmt.Printf("%s\n", pageURL)
	}
}
