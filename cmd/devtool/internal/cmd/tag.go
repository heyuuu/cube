package cmd

import (
	"fmt"
	"strings"

	"cube/cmd/devtool/internal/tag"

	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	var minor bool
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "为当前项目打下一个递增版本 tag",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTag(minor)
		},
	}
	cmd.Flags().BoolVar(&minor, "minor", false, "递增 minor 版本（默认递增 patch）")
	return cmd
}

func runTag(minor bool) error {
	root, err := rootDir()
	if err != nil {
		return err
	}
	if dirty, err := isGitDirty(root); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("工作区不干净，请先提交或暂存变更再打 tag")
	}

	last, ok, err := latestVersionTag(root)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("没有历史版本 tag，新 tag 为 v0.1.0")
		return createTag(root, tag.VersionTag{Minor: 1})
	}

	// 自上一个 tag 以来没有任何提交时无需打新 tag
	count, err := gitOut(root, "rev-list", "--count", last.String()+"..HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(count) == "0" {
		fmt.Printf("自 %s 以来没有新的提交，无需打新 tag\n", last)
		return nil
	}

	next := last.NextPatch()
	if minor {
		next = last.NextMinor()
	}
	fmt.Printf("%s -> %s\n", last, next)
	return createTag(root, next)
}

// createTag 在仓库根目录创建版本 tag
func createTag(root string, v tag.VersionTag) error {
	_, err := gitOut(root, "tag", v.String())
	return err
}

// latestVersionTag 返回最大的三段式版本 tag；没有版本 tag 时第二个返回值为 false。
func latestVersionTag(root string) (tag.VersionTag, bool, error) {
	out, err := gitOut(root, "tag", "--list")
	if err != nil {
		return tag.VersionTag{}, false, err
	}
	var latest tag.VersionTag
	found := false
	for _, line := range strings.Split(out, "\n") {
		v, ok := tag.ParseVersionTag(strings.TrimSpace(line))
		if !ok {
			continue
		}
		if !found || v.Compare(latest) > 0 {
			latest, found = v, true
		}
	}
	return latest, found, nil
}
