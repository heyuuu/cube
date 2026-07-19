package cmd

import (
	"log"

	"github.com/heyuuu/cube/internal/app"
	"github.com/heyuuu/cube/internal/util/easycobra"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &easycobra.Command{
	Use:   "server",
	Short: `run the server`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		server := app.Default().Server()

		err := server.StartHTTP(":8080")
		if err != nil {
			log.Fatal(err)
		}
	},
}
