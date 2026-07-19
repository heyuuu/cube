package alfred

import (
	"encoding/json"
	"fmt"

	"github.com/heyuuu/cube/cmd/util/easycobra"
	"github.com/heyuuu/cube/util/slicekit"
)

var RootCmd = &easycobra.Command{
	Use: "alfred",
	Children: []*easycobra.Command{
		projectSearchCmd,
		projectOpenCmd,
		appSearchCmd,
	},
}

// helpers

type H map[string]any

// see: https://www.alfredapp.com/help/workflows/inputs/script-filter/json/
type Item struct {
	Title    string `json:"title"`
	SubTitle string `json:"subtitle"`
	Arg      string `json:"arg"`
}

func PrintResult[T any](items []T, fn func(item T) Item) error {
	result := H{
		"items": slicekit.Map(items, fn),
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))
	return nil
}
