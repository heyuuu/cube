package workspace

import (
	"fmt"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/cmd/util/easycobra"
)

var RootCmd = &easycobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
	Children: []*easycobra.Command{
		workspaceListCmd,
	},
}

// cmd `workspace list`
var workspaceListCmd = &easycobra.Command{
	Use: "list",
	Run: func(args []string) error {
		service := app.Default().ProjectService()
		for _, ws := range service.Workspaces() {
			fmt.Println(ws.Name())
		}
		return nil
	},
}
