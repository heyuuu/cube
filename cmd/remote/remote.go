package remote

import (
	"fmt"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/cmd/util/console"
	"github.com/heyuuu/cube/cmd/util/easycobra"
	"github.com/heyuuu/cube/project"
)

var RootCmd = &easycobra.Command{
	Use: "remote",
	Children: []*easycobra.Command{
		remoteListCmd,
	},
}

// cmd `remote list`
var remoteListCmd = &easycobra.Command{
	Use:   "list",
	Short: "列出可用远端仓库列表",
	Run: func(args []string) error {
		service := app.Default().ProjectService()
		remotes := service.Remotes()
		showRemotes(remotes)
		return nil
	},
}

func showRemotes(remotes []*project.Remote) {
	console.PrintTableFunc(remotes, []string{
		fmt.Sprintf("Remote(%d)", len(remotes)),
		"Path",
	}, func(r *project.Remote) []string {
		return []string{
			r.Name(),
			r.Host(),
		}
	})
}
