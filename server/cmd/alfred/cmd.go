// Package alfred 提供 `cube alfred` 命令组，输出 Alfred Script Filter JSON，
// 供 Alfred workflow 集成（项目搜索、opener 搜索、项目打开）。
//
// 各业务子命令按「一命令一文件」组织（project_search.go / opener_search.go / project_open.go），
// 本文件只放命令组入口 NewCmd；通用的 Alfred JSON 输出 helper 在 helpers.go。
package alfred

import (
	"github.com/spf13/cobra"

	"cube/cmd/env"
)

// NewCmd 是 `cube alfred` 命令组入口，纯分发。
func NewCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "alfred",
		Hidden: true,
	}

	cmd.AddCommand(newProjectSearchCmd(env))
	cmd.AddCommand(newProjectOpenCmd(env))
	cmd.AddCommand(newOpenerSearchCmd(env))

	return cmd
}
