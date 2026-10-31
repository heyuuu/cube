package cmd

import (
	"encoding/json/jsontext"
	"fmt"

	"github.com/heyuuu/x/store"
	"github.com/spf13/cobra"

	"cube/internal/app"
)

func newOpenapiCmd(a *app.App) *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:     "openapi",
		Aliases: []string{"api"},
		Short:   "生成 OpenAPI 3.1 spec 到文件",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return writeOpenAPIFile(a, outPath)
		},
	}

	cmd.Flags().StringVarP(&outPath, "out", "o", "openapi.json", "输出文件路径")
	return cmd
}

// writeOpenAPIFile 生成 OpenAPI 3.1 spec，格式化后写入 outPath。
func writeOpenAPIFile(a *app.App, outPath string) error {
	data, err := a.Server().OpenAPIJSON()
	if err != nil {
		return fmt.Errorf("生成 OpenAPI 失败: %w", err)
	}

	outPath, err = ExtendPath(outPath)
	if err != nil {
		return fmt.Errorf("解析输出文件绝对路径失败: %w", err)
	}
	// spec 以 jsontext.Value 交给 SaveJSON：2 空格缩进便于人工检查（HTTP 端点仍用 compact），
	// marshal 时顺带完成语法校验
	if err := store.SaveJSON(outPath, jsontext.Value(data)); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	fmt.Printf("已生成 OpenAPI spec: %s\n", outPath)
	return nil
}
