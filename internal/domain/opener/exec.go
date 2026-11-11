package opener

import (
	"maps"
	"strings"

	"cube/internal/domain/icons"
)

// actionOpener 打开方式的唯一实现：按 role 各配一条动作串（`exec:` 命令 / `url:` 链接）。
// 纯数据实体（环境不进实体）：exec 动作构造时预分词（token[0]=可执行文件，$0/$1...
// 占位路径槽位可嵌在 token 内），url 动作渲染占位符（站内路由 query encode + 拼 baseURL）；
// 只做纯计算（BuildArgs 组装 bin+args），启动子进程的执行环境（executor）留在 Service。
type actionOpener struct {
	name    string          // 应用名, 唯一标识符
	title   string          // 展示文案，构造时已解析默认值
	actions map[Role]string // role → 动作模板；声明了哪些 role 即键集合
	icon    icons.Icon      // 图标声明，零值未配置
}

var _ Opener = (*actionOpener)(nil)

// InitActionOpener 从存储形状构造 actionOpener。
//   - actions 必填非空：键须为合法 role；每条动作串须为合法 `<kind>:<模板>`；
//   - exec 模板须可分词出至少一个 token（token[0] 是可执行文件）；
//   - url 模板须以 /（站内）或 http(s)://（外部）开头；
//   - 每条模板的占位符索引不得越该 role 的槽个数（逐条局部校验，role 间无一致性约束）；
//   - baseURL 供站内路由拼接（空串时站内动作构造报错，装配层负责注入）。
func InitActionOpener(spec Spec) (Opener, error) {
	spec, err := validateSpec(spec)
	if err != nil {
		return nil, err
	}

	return &actionOpener{
		name:    spec.Name,
		title:   spec.Title,
		actions: maps.Clone(spec.Actions),
		icon:    *spec.Icon,
	}, nil
}

func (o *actionOpener) Name() string     { return o.name }
func (o *actionOpener) Title() string    { return o.title }
func (o *actionOpener) Icon() icons.Icon { return o.icon }

// Roles 声明的 role 列表（按枚举固定序），= actions 的键集合。
func (o *actionOpener) Roles() []Role {
	roles := make([]Role, 0, len(o.actions))
	for _, r := range RoleOrder() {
		if _, ok := o.actions[r]; ok {
			roles = append(roles, r)
		}
	}
	return roles
}

// Actions 各 role 的动作串原文（编辑表单回显用）。
func (o *actionOpener) Actions() map[Role]string {
	return maps.Clone(o.actions)
}

// Summary 展示串（CLI 表格 / alfred 副标题 / Web DTO）：
// 按 role 固定序拼接 "role:动作串"。
func (o *actionOpener) Summary() string {
	parts := make([]string, 0, len(o.actions))
	for _, r := range o.Roles() {
		parts = append(parts, string(r)+":"+o.actions[r])
	}
	return strings.Join(parts, "; ")
}
