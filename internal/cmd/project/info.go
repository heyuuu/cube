package project

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/heyuuu/x/xstd/xfilepath"
	"github.com/heyuuu/x/xstd/xslices"
	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
	"cube/internal/domain/project/snapshot"
	"cube/pkg/git"
	"cube/pkg/tui"
)

// info 输出的样式组。键名青色与 PrintTable 表头同源（color 51），
// 次要信息灰色、状态色绿/黄，保持 CLI 整体视觉一致。
var (
	infoKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))            // 键名
	infoDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))           // 次要信息（网页地址、绝对时间等）
	infoWarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))           // 需注意状态（有改动、无缓存等）
	infoBoldStyle    = lipgloss.NewStyle().Bold(true)                                  // 强调（项目名）
	infoSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")) // -v 小节标题（与表格边框同色）
)

// infoKeyWidth 是键名列宽度（含对齐），多行值的续行缩进也依赖它。
const infoKeyWidth = 16

// newInfoCmd `cube project info` 命令。
func newInfoCmd(a *app.App) *cobra.Command {
	var verbose bool
	var refresh bool
	cmd := &cobra.Command{
		Use:   "info [query]",
		Short: "项目详情(支持项目名或项目路径模糊搜索)",
		Long: `显示单个项目的详情。

query 支持两种搜索模式：
  - 项目名称搜索：按关键词模糊匹配项目名称（默认）。
  - 项目路径搜索：当 query 以 '.'、'~' 或 '/' 开头时触发，
    搜索给定路径及其所有子目录中的项目。

-v 显示扩展信息（实时读仓库，不依赖缓存快照）：
  - 未提交的文件列表（类似 git status --short）。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := xslices.Get(args, 0)

			// 匹配项目
			proj, err := pickProject(a.ProjectService(), query, a.Local())
			if err != nil {
				return err
			}

			// 获取参数绝对路径，后面显示用
			var queryPath string
			if absPath, ok := cmdutil.AsPathQuery(query); ok {
				queryPath = absPath
			}

			// 基础信息
			printInfoSection("基础信息")
			printInfoLine("project", infoBoldStyle.Render(proj.Name()))
			printInfoLine("path", proj.Path())
			printInfoLine("group", cmp.Or(proj.Group(), "-"))
			printInfoLine("tags", cmp.Or(proj.TagsStr(), "-"))

			// 项目快照，默认走缓存，--refresh 强制采集最新信息
			var snap *snapshot.Entry
			if !refresh {
				snap = a.ProjectService().Snapshot(proj.Path())
				if snap == nil {
					printInfoLine("snapshot", infoWarnStyle.Render("无缓存(可增加 -r｜--refresh 参数强制采集)"))
					return nil
				}
			} else {
				snap, err = a.ProjectService().RefreshSnapshot(proj.Path())
				if err != nil {
					return err
				}
			}

			// 补充 snap 中的基础信息
			printInfoLineEx("snapshot", cmdutil.PrettyTime(snap.UpdatedAt), snap.UpdatedAt.Format(time.TimeOnly))
			printInfoLine("defaultBranch", snap.DefaultBranch)
			printInfoLine("branches", fmt.Sprintf("%d 个", len(snap.Branches)))

			// 远程信息
			printInfoSection("Remotes")
			for _, r := range snap.Remotes {
				var webURL string
				if u, err := git.ParseRepoURL(r.URL); err == nil {
					webURL = u.WebURL()
				}
				printInfoLineEx(r.Name, r.URL, webURL)
			}

			// branch upstreams
			printInfoSection("分支同步情况")
			for _, br := range snap.Branches {
				if br.Upstream == "" {
					continue
				}
				printInfoLine(br.ShortName, formatAheadBehind(br.Ahead, br.Behind))
			}

			// worktree
			printInfoSection("Worktrees")
			for wt := range snap.AllWorktrees() {
				label := "<main>"
				if !wt.Main {
					label = wt.Branch
				}
				// 在当前 worktree 前增加标识
				if queryPath != "" && xfilepath.HasPrefix(queryPath, wt.Path) {
					label = "* " + label
				}

				// 组装值
				branch := cmp.Or(wt.Branch, "HEAD detached")
				value := cmdutil.PrettyPath(wt.Path) + infoDimStyle.Render(" ("+branch+")")

				printInfoLine(label, value)
			}

			if verbose {
				printInfoVerbose(proj.Path())
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示扩展信息：未提交文件列表（实时读取）")
	cmd.Flags().BoolVarP(&refresh, "refresh", "r", false, "是否刷新 snapshot 信息")
	return cmd
}

// printInfoSection 打印 -v 的小节标题（前置空行 + 加粗着色标题）。
func printInfoSection(title string) {
	tui.Print("\n" + infoSectionStyle.Render(title) + "\n")
}

// printInfoLine 按 info 命令的对齐格式打印一行 key: value。
func printInfoLine(key, value string) {
	printInfoLineEx(key, value, "")
}

// printInfoLineEx 按 info 命令的对齐格式打印一行 key: value (dim)。
func printInfoLineEx(key, value, dim string) {
	if dim != "" {
		value += " " + infoDimStyle.Render("("+dim+")")
	}
	tui.Print(infoKeyStyle.Width(infoKeyWidth).Render(key) + ": " + value + "\n")
}

// formatAheadBehind 把 (ahead, behind) 格式化为一格内容：
//   - 全 0 → "✓"
//   - 否则形如 "+3/-1"；ahead 或 behind 为 0 时省略该半边（"+3" / "-1"）
func formatAheadBehind(ahead, behind int) string {
	if ahead == 0 && behind == 0 {
		return "✓"
	}
	var parts []string
	if ahead > 0 {
		parts = append(parts, fmt.Sprintf("+%d", ahead))
	}
	if behind > 0 {
		parts = append(parts, fmt.Sprintf("-%d", behind))
	}
	return strings.Join(parts, "/")
}

// printInfoVerbose 打印 -v 的扩展段：未提交文件。
// 数据实时读本地仓库，不依赖缓存快照；
// 读失败按 info 的容错基调降级为警告行，不中断整体输出。
func printInfoVerbose(repoPath string) {
	// ── 未提交改动：工作区干净时整段不显示 ──
	st, err := git.LoadRepoStatus(repoPath)
	if err != nil {
		printInfoSection("未提交改动")
		tui.Print(infoWarnStyle.Render(fmt.Sprintf("读取工作区状态失败: %v", err)) + "\n")
	} else if len(st.Files) > 0 {
		printInfoSection("未提交改动")
		for _, f := range st.Files {
			tui.Print(formatInfoFileLine(f) + "\n")
		}
	}
}

// formatInfoFileLine 渲染一行未提交文件状态：XY 码 + 路径。
// 有实际改动（M/A/D/R/U）黄色提示；未跟踪（??）灰色弱化。
func formatInfoFileLine(f git.FileStatus) string {
	if strings.Contains(f.Code, "?") {
		return infoDimStyle.Render(f.Code) + " " + f.Path
	}
	return infoWarnStyle.Render(f.Code) + " " + f.Path
}
