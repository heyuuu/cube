package project

import (
	"os"
	"strings"
)

// 扫描附加 tag：标记项目额外特征（git 前提假定满足，不打冗余 git 标签，见 AGENTS.md）。
const (
	TagGodot  = "godot" // Godot 项目（含 project.godot）
	TagGolang = "golang"
	TagNode   = "node"
)

func smellTags(projectPath string) []string {
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return nil
	}

	var tags []string
	for _, entry := range entries {
		switch entry.Name() {
		case "project.godot":
			tags = append(tags, TagGodot)
		case "go.mod":
			tags = append(tags, TagGolang)
		case "package.json":
			tags = append(tags, TagNode)
		}
	}

	return tags
}

// normalizeTags 清洗标签列表：trim、去空串、按序去重；全空返回 nil。
func normalizeTags(tags []string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}
