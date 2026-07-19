package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/util/easycobra"
	"github.com/heyuuu/cube/version"
)

// versionCmd represents the version command
var versionCmd = &easycobra.Command{
	Use:   "version",
	Short: "show version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("go-cube " + version.Version)
	},
}
