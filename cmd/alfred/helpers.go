package alfred

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/heyuuu/cube/util/slicekit"
)

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

func passthruRun(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
