package workspace

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/heyuuu/x/store"
	"github.com/heyuuu/x/xstd/xos"
	"go.yaml.in/yaml/v3"

	"cube/internal/domain/project/cubefile"
)

// Resolve 解析项目根（或 worktree 根）的 workspace 列表，返回相对该根的成员目录。
func Resolve(root string) []Workspace {
	return ResolveRules(root, cubefile.LoadWorkspaceRules(root))
}

// ResolveRules 在指定目录下根据给定规则列表匹配 workspace
func ResolveRules(root string, rules []cubefile.WorkspaceRule) []Workspace {
	result := make([]Workspace, 0, len(rules))
	for _, rule := range rules {
		result = append(result, resolveRule(root, rule)...)
	}
	return result
}

// ResolveRules 在指定目录下根据给定规则匹配 workspace, 跳过各种错误
// path 支持 glob 通配（apps/*、packages/** 等，语义与 npm/pnpm 一致：doublestar 展开、排除隐藏与 node_modules 目录）
func resolveRule(root string, rule cubefile.WorkspaceRule) []Workspace {
	rel, ok := cleanRulePath(root, rule.Path)
	if !ok {
		return nil
	}

	// 特殊 rule
	switch rule.Path {
	case cubefile.NpmRule:
		return resolveNpmRule(root, rule.Name)
	case cubefile.PnpmRule:
		return resolvePnpmRule(root, rule.Name)
	}

	// 模糊匹配
	if hasGlobMeta(rel) {
		return resolvePatterns(root, rule.Name, []string{rel})
	}

	// 精确匹配
	if !xos.IsDir(filepath.Join(root, rel)) {
		slog.Warn("cube.json workspace 条目目录不存在，跳过", "root", root, "path", rule.Path)
		return nil
	}

	return []Workspace{
		{Name: rule.Name, Path: rel},
	}
}

// cleanRulePath 条目 path 的公共校验：非空、相对、不逃逸出根。
// 返回清理后的相对路径；坏路径返回 ok=false（已记日志）。
func cleanRulePath(root, path string) (string, bool) {
	if path == "" {
		slog.Warn("cube.json workspace 条目缺 path，跳过", "root", root)
		return "", false
	}
	if filepath.IsAbs(path) {
		slog.Warn("cube.json workspace 条目 path 须为相对路径，跳过", "root", root, "path", path)
		return "", false
	}
	rel := filepath.Clean(path)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		slog.Warn("cube.json workspace 条目 path 逃逸出项目根，跳过", "root", root, "path", path)
		return "", false
	}
	return rel, true
}

// hasGlobMeta 判断是否含 glob 通配符（进入通配展开分支而非单目录直查）。
func hasGlobMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

// resolvePatterns 处理给定规则匹配的情况
func resolvePatterns(root string, name string, patterns []string) []Workspace {
	if len(patterns) == 0 {
		return nil
	}
	rootRel := "."
	var workspaces []Workspace
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 单目录读失败（权限等）跳过，不中断整次探测
		}
		if !d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil || rel == rootRel {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))
		if depth >= maxGlobWalkDepth {
			return fs.SkipDir
		}
		if isGlobDir(rel) {
			return fs.SkipDir // node_modules / .git 等不进 glob 结果也不下钻
		}
		for _, p := range patterns {
			if p == "" {
				continue
			}
			// doublestar 的 "**" 可匹配零段，libs/** 会连 libs 本身也命中——
			// 模式自身的静态前缀是「容器目录」不是成员，排除
			if prefix, ok := strings.CutSuffix(filepath.ToSlash(p), "/**"); ok && filepath.ToSlash(rel) == prefix {
				continue
			}
			if ok, _ := doublestar.Match(filepath.ToSlash(p), filepath.ToSlash(rel)); ok {
				workspaces = append(workspaces, Workspace{Name: name + ":" + filepath.ToSlash(rel), Path: rel})
				break
			}
		}
		return nil
	})
	return workspaces
}

// resolveNpmRule 解析 npm 的 workspaces 规则
// npm 使用 package.json 的 workspaces 节表示工作空间规则，支持数组或 {packages: []} 两种形态
func resolveNpmRule(root string, name string) []Workspace {
	// 读取原始节
	pkgFile := filepath.Join(root, "package.json")
	raw, err := store.LoadSection[jsontext.Value](pkgFile, "workspaces")
	if err != nil || len(raw) == 0 {
		return nil
	}

	// 尝试解析为 []string
	var patterns []string
	if err := json.Unmarshal(raw, &patterns); err != nil {
		return resolvePatterns(root, name, patterns)
	}

	// 尝试解析为 {packages: []string}
	var obj struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return resolvePatterns(root, name, obj.Packages)
	}

	return nil
}

// resolveNpmRule 解析 pnpm 的 workspaces 规则
// pnpm 使用 pnpm-workspace.yaml 的 workspaces 节表示工作空间规则
func resolvePnpmRule(root string, name string) []Workspace {
	data, err := os.ReadFile(filepath.Join(root, "pnpm-workspace.yaml"))
	if err != nil {
		return nil
	}
	var spec struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil
	}
	return resolvePatterns(root, name, spec.Packages)
}
