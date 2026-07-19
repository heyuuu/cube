package alfred

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/heyuuu/cube/util/easycobra"
	"github.com/heyuuu/cube/util/slicekit"
)

// alfred root cmd
var AlfredCmd = &easycobra.Command{
	Use: "alfred",
}

func init() {
	AlfredCmd.AddCommand(projectSearchCmd)
	AlfredCmd.AddCommand(projectOpenCmd)
	AlfredCmd.AddCommand(appSearchCmd)
}

// helpers

type H map[string]any

// see: https://www.alfredapp.com/help/workflows/inputs/script-filter/json/
type Item struct {
	Title    string `json:"title"`
	SubTitle string `json:"subtitle"`
	Arg      string `json:"arg"`
}

func PrintResult(items []Item) {
	result := H{
		"items": items,
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(bytes))
}

func PrintResultFunc[T any](items []T, fn func(item T) Item) {
	listItems := slicekit.Map(items, fn)
	PrintResult(listItems)
}
