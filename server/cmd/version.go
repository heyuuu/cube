package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"cube/cmd/env"
	"cube/core/version"
)

func newVersionCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "显示当前版本号",
		Long:  `显示当前版本号`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(version.AppName + ": " + version.VersionInfo())
			return nil
		},
	}
	return cmd
}
