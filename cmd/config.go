package cmd

import (
	"fmt"
	"github.com/heyuuu/go-cube/internal/config"
	"github.com/heyuuu/go-cube/internal/cube"
	"github.com/heyuuu/go-cube/internal/util/easycobra"

	"github.com/spf13/cobra"
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
