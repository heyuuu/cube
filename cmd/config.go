package cmd

import (
	"fmt"

	"github.com/heyuuu/cube/cmd/util/easycobra"
	"github.com/heyuuu/cube/config"
	"github.com/heyuuu/cube/version"
)

// configCmd represents the version command
var configCmd = &easycobra.Command{
	Use:   "config",
	Short: "show config",
	Run: func(args []string) error {
		fmt.Println("go-cube " + version.Version)
		fmt.Println("config path: " + config.ConfigPath())
		return nil
	},
}
