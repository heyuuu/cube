package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"cube/internal/version"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "显示当前版本号",
		Long:  `显示当前版本号`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(version.AppName + ": " + version.Info())
			return nil
		},
	}
	return cmd
}
