package cmd

import (
	"fmt"

	"github.com/heyuuu/cube/cmd/util/easycobra"
	"github.com/heyuuu/cube/version"
)

// versionCmd represents the version command
var versionCmd = &easycobra.Command{
	Use:   "version",
	Short: "show version",
	Run: func(args []string) error {
		fmt.Println("cube " + version.Version)
		return nil
	},
}
