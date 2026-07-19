package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/cube"
	"github.com/heyuuu/cube/util/easycobra"
)

// configCmd represents the version command
var configCmd = &easycobra.Command{
	Use:   "config",
	Short: "show config",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("go-cube " + cube.Version)
		fmt.Println("config path: " + config.ConfigPath())
	},
}
