package opener

import (
	"fmt"
	"strings"

	"github.com/heyuuu/x/xstd/xslices"
)

// Intent 打开意图（场景枚举，开放增长）：defaults（openerIntents 节）的键空间，
// 由调用点按发起场景选定。每个 intent 必然映射唯一 role（intent 的动作模板与
// 参数校验都由该 role 承载）；反之一个 role 可被多个 intent 复用
// （terminal/git/workbench 都复用 open-dir 的动作声明）。
//
// 新增 intent 是小代码变更（枚举 + role 映射），不要求任何 opener 配置跟着改——
// 想被新场景选中的 opener 只需在 openerIntents 节配置默认。
type Intent string

const (
	// IntentDir 通用打开目录：项目列表/目录树目录节点、open、open-path 目录。
	IntentDir Intent = "dir"
	// IntentFile 通用打开文件：目录树文件节点、open-path 文件。
	IntentFile Intent = "file"
	// IntentDiffDir 对比两个目录：cube diff。
	IntentDiffDir Intent = "diff-dir"
	// IntentDiffFile 对比两个文件：cube diff、diff 面板外部对比。
	IntentDiffFile Intent = "diff-file"
	// IntentTerminal 在终端打开目录。
	IntentTerminal Intent = "terminal"
	// IntentGit 在 git 客户端打开仓库。
	IntentGit Intent = "git"
	// IntentWorkbench 在 cube 工作台打开。
	IntentWorkbench Intent = "workbench"
	// IntentIde 在 IDE 打开目录。
	IntentIde Intent = "ide"
	// IntentDoc 文档查看打开（md 页「作为文档打开」等）。
	IntentDoc Intent = "doc"
)

// intentOrder intent 的固定展示序。
var intentOrder = []Intent{
	IntentDir, IntentFile, IntentDiffDir, IntentDiffFile,
	IntentTerminal, IntentGit, IntentWorkbench, IntentIde, IntentDoc,
}

// intentRoles intent → role 的唯一映射（多对一）。
var intentRoles = map[Intent]Role{
	IntentDir:       RoleOpenDir,
	IntentFile:      RoleOpenFile,
	IntentDiffDir:   RoleDiffDir,
	IntentDiffFile:  RoleDiffFile,
	IntentTerminal:  RoleOpenDir,
	IntentGit:       RoleOpenDir,
	IntentWorkbench: RoleOpenDir,
	IntentIde:       RoleOpenDir,
	IntentDoc:       RoleOpenFile,
}

// Role 返回该 intent 对应的 role（动作模板与槽校验的承载）。
func (i Intent) Role() Role { return intentRoles[i] }

// ParseIntent 解析 intent 字符串，未知值返回中文错误。
func ParseIntent(s string) (Intent, error) {
	i := Intent(strings.TrimSpace(s))
	if _, ok := intentRoles[i]; !ok {
		return "", fmt.Errorf("未知的 opener intent %q（合法值：%s）", s, strings.Join(intentNames(), ","))
	}
	return i, nil
}

func intentNames() []string {
	parts := make([]string, len(intentOrder))
	for i, it := range intentOrder {
		parts[i] = string(it)
	}
	return parts
}

// IntentInfo intents API 的输出条目：合成后的意图状态（默认 + 候选清单）。
type IntentInfo struct {
	Intent        Intent
	DefaultOpener string
	Openers       []string
}

// 规范化处理 []IntentInfo，仅保留合法值
func normalizeIntentInfos(infos []IntentInfo, openers []Opener) (result []IntentInfo, changed bool) {
	openerMap := xslices.ToMap(openers, func(o Opener) (string, *Opener) { return o.Name, &o })
	// 检验 openerName 是否有效，如果无效则修改为空字符串
	normalizeOpener := func(intent Intent, openerName string) string {
		if openerName == "" {
			return ""
		}
		if o := openerMap[openerName]; o != nil && o.SupportsRole(intent.Role()) {
			return openerName
		}

		changed = true
		return ""
	}

	result = make([]IntentInfo, len(infos))
	for i, info := range infos {
		intent := info.Intent
		result[i] = IntentInfo{
			Intent:        intent,
			DefaultOpener: normalizeOpener(intent, info.DefaultOpener),
			Openers: xslices.FilterMap(info.Openers, func(openerName string) (string, bool) {
				openerName = normalizeOpener(intent, openerName)
				return openerName, openerName != ""
			}),
		}
	}
	return result, changed
}
