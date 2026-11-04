package project

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"cube/pkg/iconkit"
	"cube/pkg/x"
)

// ScanRule 扫描规则
type ScanRule struct {
	Group    string        `json:"group"`          // 扫描出的项目组名
	Path     string        `json:"path"`           // 扫描的根目录
	MaxDepth int           `json:"maxDepth"`       // 最大扫描深度（根下直接子项为 0，-1 不限深）
	Icon     *iconkit.Icon `json:"icon,omitempty"` // 组图标（可选，语义见 pkg/iconkit）
	Tags     []string      `json:"tags,omitempty"` // 命中该规则的项目统一追加的标签（如 个人/公司/开源，可选）
}

func scan(rules []ScanRule) ([]*Project, error) {
	var all []*Project
	for _, rule := range rules {
		got, err := scanOne(rule)
		if err != nil {
			return nil, err
		}
		all = append(all, got...)
	}
	return all, nil
}

func scanOne(r ScanRule) ([]*Project, error) {
	root, maxDepth := filepath.Clean(r.Path), r.MaxDepth

	var projects []*Project
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// 检查路径 name 是否应跳过
		// 特例: 规则根本身以 "." / "_" 开头时不会跳过
		if p != root && skipDirName(d.Name()) {
			return fs.SkipDir
		}

		// 项目目录：收录且不再下潜（项目内部的嵌套仓库不作为独立项目）。
		if isProjectPath(p) {
			projects = append(projects, newProject(r, p, slices.Concat(r.Tags, smellTags(p))))
			return fs.SkipDir
		}

		// 深度剪枝：根下直接子项为深度 0，到达 maxDepth 后只访问不下潜；
		// 根自身（rel == "."）不受限，-1 不限深
		if maxDepth >= 0 {
			if rel, relErr := filepath.Rel(root, p); relErr == nil && rel != "." &&
				strings.Count(rel, string(filepath.Separator)) >= maxDepth {
				return fs.SkipDir
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return projects, nil
}

// skipDirName 判断目录名是否被扫描跳过（以 "." 或 "_" 开头）。
func skipDirName(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// 判断是否为项目目录
func isProjectPath(path string) bool {
	// 以 .git 目录是否存在判定
	// 注意，判断的是 .git 目录；git worktree 目录会有 .git 文件，但是 git worktree 目录不作为单独项目
	gitPath := filepath.Join(path, ".git")
	return x.IsDir(gitPath)
}

// MatchScanRule 判断 absPath（git init 后）能否被 scan 收录为新项目，
// 返回匹配的规则及对应项目名。多条规则命中时取根路径最长的一条（最内层规则），
// 与 MatchCloneRule 的最长前缀取舍一致。
//
// 判定条件与 scanOne 的遍历语义一致：
//   - absPath 位于规则根目录之下（或即根目录本身），相对深度 <= maxDepth
//     （根下直接子项为深度 0，-1 不限深）
//   - 规则根下到 absPath 的每一级目录名均不以 "." 或 "_" 开头——否则遍历
//     在上级就被剪掉；规则根本身不受该前缀限制（scanOne 同此特例）
func MatchScanRule(absPath string, rules []ScanRule) (rule ScanRule, name string, ok bool) {
outer:
	for _, r := range rules {
		rel, err := filepath.Rel(r.Path, absPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue // absPath 不在该规则根目录之下
		}

		// rel != "." 时才有中间层级可检查；rel == "." 即规则根自身（深度 0）
		if rel != "." {
			// 深度检查：直接子目录为深度 0，超出 maxDepth 的路径遍历不可达（-1 不限深）
			segments := strings.Split(rel, string(filepath.Separator))
			if r.MaxDepth >= 0 && len(segments)-1 > r.MaxDepth {
				continue
			}

			// 前缀检查：规则根到 absPath 的每一级目录都不得以 "." / "_" 开头，
			// 否则遍历在上级就被剪掉
			for _, segment := range segments {
				if skipDirName(segment) {
					continue outer
				}
			}
		}

		if !ok || len(r.Path) > len(rule.Path) {
			rule, name, ok = r, projectName(r, absPath), true
		}
	}
	return
}
