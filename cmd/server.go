package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/util/easycobra"
)

// serverCmd represents the server command
var serverCmd = &easycobra.Command{
	Use:   "server",
	Short: `run the server`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		server := app.Default().Server()

		err := server.Start(":8080")
		if err != nil {
			log.Fatal(err)
		}
	},
}
