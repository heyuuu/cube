package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示最后的版本 tag",
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := rootDir()
			if err != nil {
				return err
			}
			last, ok, err := latestVersionTag(root)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("当前仓库还没有版本 tag")
			}
			fmt.Println(last)
			return nil
		},
	}
}
