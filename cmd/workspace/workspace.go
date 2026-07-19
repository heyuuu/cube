package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/util/easycobra"
)

var WorkspaceCmd = &easycobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
}

func init() {
	WorkspaceCmd.AddCommand(workspaceListCmd)
}

// cmd `workspace list`
var workspaceListCmd = &easycobra.Command{
	Use: "list",
	Run: func(cmd *cobra.Command, args []string) {
		service := app.Default().WorkspaceService()
		for _, ws := range service.Workspaces() {
			fmt.Println(ws.Name())
		}
	},
}
