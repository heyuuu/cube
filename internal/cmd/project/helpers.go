package project

import (
	"errors"
	"fmt"

	"cube/internal/cmd/cmdutil"
	"cube/internal/domain/project"
	"cube/pkg/tui"
)

// pickProject cmdutil.PickProject 的别名
// pickProject 理论上是 cmd/project 包内函数，但是放在了 cmd/cmdutil 方便共享；此处留一个别名在包内使用
func pickProject(service *project.Service, query string, localMode bool) (*project.Project, error) {
	return cmdutil.PickProject(service, query, localMode)
}

// pickOpenTarget 选择项目的打开目标（根目录 / workspaces / worktrees，1032+1030）：
// 单目标直接返回主目录；多目标交互选择，标签为 "main" 或 worktree 分支名（可带 workspace 名）。
func pickOpenTarget(service *project.Service, proj *project.Project) (string, error) {
	snap := service.Snapshot(proj.Path())
	if snap == nil {
		return proj.Path(), nil
	}

	targets := cmdutil.OpenTargets(snap)
	if len(targets) <= 1 {
		return proj.Path(), nil
	}

	pick, err := tui.SelectItem("选择打开目标", targets, func(t cmdutil.OpenTarget) string { return t.Label })
	if err != nil {
		if errors.Is(err, tui.ErrNotTTY) {
			return "", fmt.Errorf("该项目有 %d 个打开目标（主目录 + worktree + workspace），非 TTY 环境无法交互选择: %s", len(targets), proj.Path())
		}
		if errors.Is(err, tui.ErrUserAborted) {
			return "", fmt.Errorf("用户取消了打开目标选择: %w", err)
		}
		return "", err
	}
	return pick.Path, nil
}
