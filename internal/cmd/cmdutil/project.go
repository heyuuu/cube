package cmdutil

import (
	"errors"
	"fmt"

	"cube/internal/domain/project"
	"cube/pkg/pathkit"
	"cube/pkg/tui"
	"cube/pkg/x"
)

// IsPathQuery 判断 query 是否为路径(以`.`/`~`/`/` 开头时，当做路径)
func IsPathQuery(query string) bool {
	return len(query) > 0 && (query[0] == '.' || query[0] == '~' || query[0] == '/')
}

// SearchProjects 搜索项目列表
//
// query 为搜索关键词，默认为搜索项目名；当以`.`/`~`/`/` 开头时，当做路径。
// 路径 query 在本层用 ExtendPath 基于 cwd 解析为绝对路径后再传入 domain——
// cwd 依赖属于出口层职责，domain 只接受绝对路径/~ 前缀。
// upper 表示是否向上搜索。仅 query 为路径时生效，用于在项目子目录标定当前目录时使用。
func SearchProjects(service *project.Service, query string, up bool) ([]*project.Project, error) {
	if IsPathQuery(query) {
		absPath, err := ExtendPath(query)
		if err != nil {
			return nil, fmt.Errorf("解析路径 query 失败: query=%s err=%w", query, err)
		}

		return service.SearchByPath(absPath, up), nil
	}
	return service.SearchByName(query), nil
}

// PickProject 根据关键词匹配项目：精确匹配直接返回，多项匹配则交互选择。
//
// 非交互环境不支持多项选择，会报错提示使用精确名称或路径。
// --local 模式（p --local 模式）下 query 缺省视作 "."，以 cwd 为起点定位项目；
// 显式给了 query 则不干预，--local 对其无效果。
//
// 路径 query 命中 worktree 目录时归并到主项目（1032：worktree 不再是独立项目，
// pull / push / info 等在 worktree 内执行等同于操作主仓库根目录）。
func PickProject(service *project.Service, query string, localMode bool) (*project.Project, error) {
	if localMode && query == "" {
		query = "."
	}
	projects, err := SearchProjects(service, query, true)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		// 路径 query 落在 worktree 内：SearchByPath 找不到（worktree 通常在项目目录之外），归并主项目
		if IsPathQuery(query) {
			if absPath, err := ExtendPath(query); err == nil {
				if proj := service.ResolveProject(absPath); proj != nil {
					return proj, nil
				}
			}
		}
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
		x.Map(projects, func(p *project.Project) []string {
			repoURL := ""
			if info, ok := service.GitInfo(p.Path()); ok {
				repoURL = info.RepoURL
			}
			return []string{
				p.Name(),
				pathkit.PrettyPath(p.Path()),
				repoURL,
			}
		}),
	)
}
