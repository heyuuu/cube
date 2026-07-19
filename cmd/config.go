package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/util/easycobra"
	"github.com/heyuuu/cube/version"
)

// configCmd represents the version command
var configCmd = &easycobra.Command{
	Use:   "config",
	Short: "show config",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("go-cube " + version.Version)
		fmt.Println("config path: " + config.ConfigPath())
	},
}
