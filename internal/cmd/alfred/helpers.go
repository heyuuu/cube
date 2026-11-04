package alfred

import (
	"encoding/json/v2"
	"fmt"

	"github.com/heyuuu/x/xstd/xslices"
)

// H Alfred Script Filter JSON 顶层对象（字段名→值）。
type H map[string]any

// Item Alfred Script Filter 输出的单条结果。
// see: https://www.alfredapp.com/help/workflows/inputs/script-filter/json/
type Item struct {
	Title    string `json:"title"`
	SubTitle string `json:"subtitle"`
	Arg      string `json:"arg"`
}

// PrintResult 把源列表逐项映射为 Item 后输出。
func PrintResult[T any](items []T, fn func(item T) Item) error {
	return PrintItems(xslices.Map(items, fn))
}

// PrintItems 直接输出已构造好的 Item 列表（条目不与单一源列表一一对应时用，如项目平铺展开为多目标）。
func PrintItems(items []Item) error {
	result := H{
		"items": items,
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))
	return nil
}
