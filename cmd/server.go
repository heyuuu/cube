package cmd

import (
	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/cmd/util/easycobra"
)

// serverCmd represents the server command
var serverCmd = &easycobra.Command{
	Use:   "server",
	Short: `run the server`,
	Args:  cobra.NoArgs,
	Run: func(args []string) error {
		server := app.Default().Server()

		return server.Start(":8080")
	},
}
