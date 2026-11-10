package cmdutil

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/heyuuu/x/xstd/xos"
	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/domain/project"
	"cube/internal/domain/project/snapshot"
	"cube/pkg/tui"
)

// IsPathQuery 判断 query 是否为路径, 仅以下情况满足
// - ".", "..", "~"
// - 以 "./", "../", "~/", "/" 开头
func IsPathQuery(query string) bool {
	if len(query) == 0 {
		return false
	}

	prefix, _, _ := strings.Cut(query, "/")
	return prefix == "." || prefix == ".." || prefix == "~" || prefix == ""
}

// AsPathQuery 尝试作为路径 query 参数，成功返回 (绝对路径, true)
func AsPathQuery(query string) (string, bool) {
	if !IsPathQuery(query) {
		return "", false
	}

	// 用 ExtendPath 而非 filepath.Abs：~ 前缀需展开为 home，domain 只接受绝对路径
	abs, err := ExtendPath(query)
	if err != nil {
		slog.Warn("尝试解析 query 路径失败", "query", query, "err", err)
		return "", false
	}

	return abs, true
}

// SearchProjects 搜索项目列表
// - query 为路径时（IsPathQuery 判断）列出该路径之下的项目; 否则, query 作为项目名模糊搜索
func SearchProjects(service *project.Service, query string) []*project.Project {
	if absPath, ok := AsPathQuery(query); ok {
		return service.SearchInPath(absPath)
	}
	return service.SearchByName(query)
}

// PickProject 根据关键词匹配项目：精确匹配直接返回，多项匹配则交互选择
// - query 为路径时（IsPathQuery 判断）返回目录处于哪个项目之中; 否则, query 作为项目名模糊搜索
// - 非交互环境不支持多项选择，会报错提示使用精确名称或路径
// - --local 模式下 query 缺省视作 "."，以 cwd 为起点定位项目; 显式给了 query 则不干预，--local 对其无效果。
func PickProject(service *project.Service, query string, localMode bool) (*project.Project, error) {
	if localMode && query == "" {
		query = "."
	}

	// 按路径查找项目
	if absPath, ok := AsPathQuery(query); ok {
		if proj := service.ResolveProject(absPath); proj != nil {
			return proj, nil
		}
		return nil, fmt.Errorf("未找到匹配的 project: query=`%s`", query)
	}

	// 按项目名查找
	projects := service.SearchByName(query)
	if len(projects) == 0 {
		return nil, fmt.Errorf("未找到匹配的 project: query=`%s`", query)
	} else if len(projects) == 1 {
		return projects[0], nil
	}

	// 匹配多个 project 时，触发用户选择
	pick, err := tui.SelectItem("选择 project", projects, (*project.Project).Name)
	if err != nil {
		if errors.Is(err, tui.ErrNotTTY) {
			return nil, fmt.Errorf("非 TTY 环境请使用精确项目名或项目路径，避免匹配多项。: query=`%s`", query)
		}
		if errors.Is(err, tui.ErrUserAborted) {
			return nil, fmt.Errorf("用户取消了 project 选择: %w", err)
		}
		return nil, err
	}
	return pick, nil
}

// ShowProjects 输出项目列表表格
func ShowProjects(service *project.Service, projects []*project.Project) {
	tui.PrintTable(
		[]string{
			fmt.Sprintf("项目(%d)", len(projects)),
			"Path",
			"RepoUrl",
		},
		xslices.Map(projects, func(p *project.Project) []string {
			repoURL := ""
			if info := service.Snapshot(p.Path()); info != nil {
				repoURL = info.OriginURL()
			}
			return []string{
				p.Name(),
				PrettyPath(p.Path()),
				repoURL,
			}
		}),
	)
}

type OpenTarget struct {
	Label string
	Path  string
}

// OpenTargets 返回 OpenTarget 数组；结果不为空时，首位必然是主 worktree 根目录
func OpenTargets(snap *snapshot.Entry) []OpenTarget {
	if snap == nil {
		return nil
	}

	projPath := snap.Main.Path

	var targets []OpenTarget
	for worktree, ws := range snap.AllWorkspaces() {
		path := filepath.Join(worktree.Path, ws.Path)
		if !xos.IsDir(path) {
			continue
		}

		var label string
		if worktree.Path == projPath { // main worktree
			label = "main"
		} else { // linked worktree
			label = worktree.Branch // 使用分支来标识 worktree
		}
		if ws.Name != "" {
			label += " · " + ws.Name
		}

		targets = append(targets, OpenTarget{Label: label, Path: path})
	}
	return targets
}
