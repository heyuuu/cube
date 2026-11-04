package forge

import (
	"fmt"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/domain/forge"
	"cube/pkg/iconkit"
	"cube/pkg/tui"
	"cube/pkg/x"
)

// newListCmd `cube forge list` 命令。
func newListCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出全部 forge",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runForgeList(a)
		},
	}
	return cmd
}

func runForgeList(a *app.App) error {
	forges := a.ForgeService().Forges()
	tui.PrintTable(
		[]string{
			fmt.Sprintf("Forge(%d)", len(forges)),
			"Kind",
			"Icon",
		},
		x.Map(forges, func(f forge.Forge) []string {
			return []string{f.Host, f.Kind, iconSummary(f.Icon)}
		}),
	)
	return nil
}

// iconSummary CLI 侧 icon 概要：lucide 显图名，image 只显标记（base64 不进表格）。
func iconSummary(icon *iconkit.Icon) string {
	if icon == nil {
		return "-"
	}
	if icon.Type == iconkit.IconTypeImage {
		return "[image]"
	}
	return icon.Value
}
