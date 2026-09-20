package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"cube/app"
	"cube/cmd/env"
	"cube/core/server"
)

func newUiCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "打开 cube Web UI（首页 / md 页）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWeb(env.App())
		},
	}

	cmd.AddCommand(newMdCmd(env))
	cmd.AddCommand(newWorkbenchCmd(env))

	return cmd
}

// newMdCmd `cube ui md <path>` —— 以 Web 方式打开 markdown 文件（原一级命令 `cube md` 迁入）。
//
// 页面与渲染归前端工程（/md?path=<abs>），本命令只负责拼 URL 并开浏览器。
func newMdCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "md <path>",
		Short: "以 Web 方式打开 markdown 文件",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return openPage(env.App(), "md", args[0])
		},
	}
	return cmd
}

// newWorkbenchCmd `cube ui workbench <path>` —— 打开项目的工作台页面（与前端路由 /workbench 同名对齐）。
//
// 这是「打开工作台」这类 opener 的落地形态：opener 配置成 exec 命令
// `["cube", "ui", "workbench", "$0"]` 即可，URL 拼接（端口/路由/转义）收敛在本命令。
func newWorkbenchCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workbench <path>",
		Short: "打开工作台（workbench）页面",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return openPage(env.App(), "workbench", args[0])
		},
	}
	return cmd
}

// runWeb 打开 Web UI 首页。
func runWeb(a *app.App) error {
	client := server.NewClient(a.Server().ServerURL())
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
	absPath, err := ExtendPath(rawPath)
	if err != nil {
		return fmt.Errorf("解析路径失败: %w", err)
	}
	// 目录也放行：/md 页对目录展示左侧文件树（无 md 的目录显示空态）
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("路径不存在: %s", absPath)
	}

	client := server.NewClient(a.Server().ServerURL())
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
