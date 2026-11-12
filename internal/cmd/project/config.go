package project

import (
	"errors"
	"fmt"
	"os"

	"github.com/heyuuu/x/xstd/xslices"
	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/domain/project/cubefile"
	"cube/internal/domain/project/workspace"
	"cube/pkg/tui"
)

// newConfigCmd `cube project config` 命令：TTY 交互管理项目配置 .cube/cube.json。
func newConfigCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [query]",
		Short: "交互管理项目配置 .cube/cube.json（workspaces 规则等）",
		Long: `TTY 交互管理项目配置文件 .cube/cube.json（项目内配置，进 git 跟仓库走）。

当前支持 workspaces 规则的初始化固化、添加、移除与恢复默认；
cube.json 未来新增配置节也在此命令扩展操作入口。

query 支持项目名模糊搜索，规则同 open 命令。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			err := runConfigFlow(a, xslices.Get(args, 0))
			if errors.Is(err, tui.ErrUserAborted) {
				return nil // Ctrl+C 取消不算命令失败
			}
			return err
		},
	}
	return cmd
}

// configEnv 一次 config 会话的上下文：目标项目与服务依赖。
type configEnv struct {
	app  *app.App
	name string
	root string
}

// refreshSnapshot 写盘后定向重采集：写盘即刻反映到打开目标，不等 TTL。
func (env *configEnv) refreshSnapshot() {
	if _, err := env.app.ProjectService().RefreshSnapshot(env.root); err != nil {
		fmt.Printf("cube.json 已写入，但刷新快照失败（等下次采集自愈）: %v\n", err)
	}
}

// runConfigFlow 主流程：定位项目 → 循环（状态头 + 操作菜单）直到退出。
// 操作内的业务错误打印后回到菜单继续；交互取消（Ctrl+C）整体退出。
func runConfigFlow(a *app.App, query string) error {
	if !tui.IsTTY() {
		return errors.New("此命令需要 TTY 交互环境，非交互场景请用 Web 端项目详情的 workspace 编辑")
	}
	proj, err := pickProject(a.ProjectService(), query, a.Local())
	if err != nil {
		return err
	}
	env := &configEnv{app: a, name: proj.Name(), root: proj.Path()}

	for {
		printConfigState(env)
		action, err := selectConfigAction(env)
		if err != nil {
			return err // 菜单选择取消/失败：结束会话
		}
		if action.run == nil {
			return nil
		}
		if err := action.run(env); err != nil {
			if errors.Is(err, tui.ErrUserAborted) {
				return err // 上抛由 RunE 统一转静默退出
			}
			fmt.Printf("操作失败: %v\n", err)
		}
	}
}

// printConfigState 打印会话状态头：项目 / 配置文件 / workspaces 声明态。
func printConfigState(env *configEnv) {
	fileState := "未创建"
	if _, err := os.Stat(cubefile.FilePath(env.root)); err == nil {
		fileState = "已存在"
	}
	wsState := "未声明 —— 默认规则 <npm>+<pnpm> 生效中"
	if cubefile.WorkspaceRulesDeclared(env.root) {
		wsState = fmt.Sprintf("已显式声明 %d 条规则", len(cubefile.LoadWorkspaceRules(env.root)))
	}
	fmt.Println()
	fmt.Printf("项目: %s  (%s)\n", env.name, env.root)
	fmt.Printf("配置文件: %s（%s）\n", cubefile.FilePath(env.root), fileState)
	fmt.Printf("workspaces: %s\n", wsState)
}

// configAction 菜单操作项；run 为 nil 表示退出会话。
type configAction struct {
	label string
	run   func(env *configEnv) error
}

// selectConfigAction 展示操作菜单并返回选择。菜单按当前状态组装（label 随
// 声明态变化）；cubefile 未来新增配置节/操作时在 actions 追加条目即可——
// 本命令即 cube.json 的统一管理入口。
func selectConfigAction(env *configEnv) (configAction, error) {
	declared := cubefile.WorkspaceRulesDeclared(env.root)

	initLabel := "初始化 workspaces —— 按当前生效规则解析成员，挑选后固化"
	if declared {
		initLabel = "重新解析固化 workspaces —— 覆盖现有显式声明"
	}
	actions := []configAction{
		{label: initLabel, run: actionInitWorkspaces},
		{label: "添加 workspace 规则（目录 / glob / <npm> / <pnpm>）", run: actionAddRule},
		{label: "移除 workspace 规则", run: actionRemoveRule},
	}
	if declared {
		actions = append(actions, configAction{label: "恢复默认规则 —— 删除 workspaces 声明", run: actionResetRules})
	}
	actions = append(actions,
		configAction{label: "查看 workspaces 规则与生效成员", run: actionShowRules},
		configAction{label: "退出", run: nil},
	)

	options := make([]tui.Option[int], len(actions))
	for i, act := range actions {
		options[i] = tui.Option[int]{Label: act.label, Value: i}
	}
	idx, err := tui.Select("选择操作", options)
	if err != nil {
		return configAction{}, err
	}
	return actions[idx], nil
}

// actionInitWorkspaces 按当前生效规则解析成员，多选挑选后固化为显式声明
// （解析产的是候选全集，覆盖会丢失人工裁剪）。
func actionInitWorkspaces(env *configEnv) error {
	if cubefile.WorkspaceRulesDeclared(env.root) {
		overwrite, err := tui.Confirm("已存在显式 workspaces 规则，重新解析并覆盖？")
		if err != nil {
			return err
		}
		if !overwrite {
			fmt.Println("已取消覆盖，未修改")
			return nil
		}
	}

	// 用「与采集一致的规则」解析，保证看到的候选就是当前生效全集
	members := workspace.Resolve(env.root)
	if len(members) == 0 {
		return errors.New("未解析到 workspace 成员（默认规则读 pnpm-workspace.yaml / package.json workspaces），可改用「添加规则」手动声明")
	}

	picked, err := tui.MultiSelectItemWithDefaults(
		"挑选 workspace 成员（固化为显式声明）",
		members,
		func(w workspace.Workspace) string { return w.Name + "  (" + w.Path + ")" },
		members, // 默认全选：解析命中即认为都是成员，裁剪是减法
	)
	if err != nil {
		return err
	}
	if len(picked) == 0 {
		fmt.Println("未选择任何成员，未写入")
		return nil
	}

	if err := workspace.Save(env.root, picked); err != nil {
		return err
	}
	env.refreshSnapshot()
	fmt.Printf("已写入（%d 个 workspace），记得提交进 git\n", len(picked))
	return nil
}

// ruleKind 添加规则的类型：特殊规则带固定 path，手输类需用户输入 path。
type ruleKind struct {
	label  string
	path   string // 特殊规则固定 path；手输类为空
	manual bool   // 手输类：需要用户输入 path
}

// pickRuleKind 选择要添加的规则类型。
func pickRuleKind() (ruleKind, error) {
	kinds := []ruleKind{
		{label: "目录 / glob 通配（如 packages/*、apps/**）", manual: true},
		{label: "特殊规则 <npm> —— 读 package.json 的 workspaces 节", path: cubefile.NpmRule},
		{label: "特殊规则 <pnpm> —— 读 pnpm-workspace.yaml 的 packages", path: cubefile.PnpmRule},
	}
	return tui.SelectItem("选择规则类型", kinds, func(k ruleKind) string { return k.label })
}

// actionAddRule 追加一条 workspace 规则。未声明时以默认规则为编辑起点
// （与 Web workspace 编辑入口同语义：起点即当前生效配置）。
func actionAddRule(env *configEnv) error {
	kind, err := pickRuleKind()
	if err != nil {
		return err
	}

	rule := cubefile.WorkspaceRule{Path: kind.path}
	if kind.manual {
		path, err := tui.Input("输入规则 path（相对项目根的目录或 glob）", "", "packages/*", func(s string) error {
			return workspace.ValidateRule(cubefile.WorkspaceRule{Path: s})
		})
		if err != nil {
			return err
		}
		rule.Path = path
	}
	name, err := tui.InputInline("显示名（可选，通配/特殊规则作组前缀）", "", "回车跳过", nil)
	if err != nil {
		return err
	}
	rule.Name = name

	rules := cubefile.LoadWorkspaceRules(env.root)
	for _, r := range rules {
		if r.Path == rule.Path {
			return fmt.Errorf("规则已存在，未添加: %s", rule.Path)
		}
	}
	rules = append(rules, rule)
	if err := workspace.SaveRules(env.root, rules); err != nil {
		return err
	}
	env.refreshSnapshot()
	fmt.Printf("已添加规则 %s（清单共 %d 条），记得提交进 git\n", ruleLabel(rule), len(rules))
	return nil
}

// actionRemoveRule 从当前规则清单中多选移除；移空前确认（空清单 = 显式「无 workspace」）。
func actionRemoveRule(env *configEnv) error {
	rules := cubefile.LoadWorkspaceRules(env.root)
	if len(rules) == 0 {
		fmt.Println("当前规则清单为空，无可移除项")
		return nil
	}
	picked, err := tui.MultiSelectItem("选择要移除的规则", rules, ruleLabel)
	if err != nil {
		return err
	}
	if len(picked) == 0 {
		fmt.Println("未选择任何规则，未修改")
		return nil
	}

	removeSet := xslices.ToBoolSet(picked, func(r cubefile.WorkspaceRule) cubefile.WorkspaceRule { return r })
	remaining := xslices.Filter(rules, func(r cubefile.WorkspaceRule) bool { return !removeSet[r] })

	if len(remaining) == 0 {
		ok, err := tui.ConfirmInlineDefault("移除全部后将写入空清单（显式声明「无 workspace」），继续？", false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("已取消，未修改")
			return nil
		}
	}

	if err := workspace.SaveRules(env.root, remaining); err != nil {
		return err
	}
	env.refreshSnapshot()
	fmt.Printf("已移除 %d 条规则（清单剩 %d 条），记得提交进 git\n", len(picked), len(remaining))
	return nil
}

// actionResetRules 删除 workspaces 声明，恢复程序默认规则（<npm>+<pnpm>）。
func actionResetRules(env *configEnv) error {
	if !cubefile.WorkspaceRulesDeclared(env.root) {
		fmt.Println("当前未声明 workspaces 规则，默认规则（<npm>+<pnpm>）已在生效中")
		return nil
	}
	ok, err := tui.Confirm("删除 workspaces 声明，恢复默认规则（<npm>+<pnpm>）？")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("已取消，未修改")
		return nil
	}
	if err := workspace.Reset(env.root); err != nil {
		return err
	}
	env.refreshSnapshot()
	fmt.Println("已恢复默认规则（<npm>+<pnpm>），记得提交进 git")
	return nil
}

// actionShowRules 打印当前规则清单与成员解析预览（只读）。
func actionShowRules(env *configEnv) error {
	rules := cubefile.LoadWorkspaceRules(env.root)
	fmt.Println()
	if cubefile.WorkspaceRulesDeclared(env.root) {
		fmt.Printf("workspaces 规则（%d 条，已显式声明）:\n", len(rules))
	} else {
		fmt.Printf("workspaces 规则（%d 条，程序默认——未显式声明）:\n", len(rules))
	}
	rows := xslices.Map(rules, func(r cubefile.WorkspaceRule) []string { return []string{r.Name, r.Path} })
	tui.PrintTable([]string{"name", "path"}, rows)

	members := workspace.Resolve(env.root)
	fmt.Printf("\n成员解析预览（按当前规则解析，%d 个）:\n", len(members))
	memberRows := xslices.Map(members, func(w workspace.Workspace) []string { return []string{w.Name, w.Path} })
	tui.PrintTable([]string{"name", "path"}, memberRows)
	return nil
}

// ruleLabel 规则在列表/输出中的展示文案：有 name 时「name  (path)」，否则 path。
func ruleLabel(r cubefile.WorkspaceRule) string {
	if r.Name != "" {
		return r.Name + "  (" + r.Path + ")"
	}
	return r.Path
}
