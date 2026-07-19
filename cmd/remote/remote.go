package remote

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/entities"
	"github.com/heyuuu/cube/util/console"
	"github.com/heyuuu/cube/util/easycobra"
)

var RemoteCmd = &easycobra.Command{
	Use: "remote",
}

func init() {
	RemoteCmd.AddCommand(remoteListCmd)
}

// cmd `remote list`
var remoteListCmd = &easycobra.Command{
	Use:   "list",
	Short: "列出可用远端仓库列表",
	Run: func(cmd *cobra.Command, args []string) {
		service := app.Default().RemoteService()
		remotes := service.Remotes()
		showRemotes(remotes)
	},
}

func showRemotes(remotes []*entities.Remote) {
	console.PrintTableFunc(remotes, []string{
		fmt.Sprintf("Remote(%d)", len(remotes)),
		"Path",
	}, func(r *entities.Remote) []string {
		return []string{
			r.Name(),
			r.Host(),
		}
	})
}
