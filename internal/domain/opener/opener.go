package opener

import (
	"fmt"
	"strings"

	"cube/internal/domain/icons"
)

// Opener opener 在 settings.json openers 节内的存储形状。
// 节内结构是 opener 域的知识，DTO 归本包（settings 包不感知节内形状）。
type Opener struct {
	Name    string          `json:"name"`
	Title   string          `json:"title,omitempty"` // 展示文案（如「打开所在目录」「使用 VS Code 打开」），缺省由 name 生成
	Actions map[Role]string `json:"actions"`         // role → 动作串 `<kind>:<模板>`（kind: exec 命令 / url 链接，见 exec.go）；$0/$1... 占位路径槽位，可嵌在模板内（如 --wd=$0）。声明了哪些 role 即键集合
	Icon    *icons.Icon     `json:"icon,omitempty"`  // 图标声明（前端渲染方式），缺省无图标
}

func validateOpener(spec Opener) (Opener, error) {
	var zero Opener

	// name
	if spec.Name == "" {
		return zero, fmt.Errorf("opener name 不得为空")
	}

	// title
	if spec.Title == "" {
		spec.Title = "用 " + spec.Name + " 打开"
	}

	// actions
	if len(spec.Actions) == 0 {
		return zero, fmt.Errorf("opener %q 缺少必填字段 actions", spec.Name)
	}
	for role, raw := range spec.Actions {
		if err := ValidateRole(role); err != nil {
			return zero, err
		}
		if _, err := validateAction(role, raw); err != nil {
			return zero, err
		}
	}

	// icon
	if err := icons.ValidateIcon(spec.Icon); err != nil {
		return zero, fmt.Errorf("opener %q icon 配置错误: %w", spec.Name, err)
	}

	return spec, nil
}

// Roles 声明的 role 列表（按枚举固定序）
func (o Opener) Roles() []Role {
	roles := make([]Role, 0, len(o.Actions))
	for _, r := range allRoles {
		if _, ok := o.Actions[r]; ok {
			roles = append(roles, r)
		}
	}
	return roles
}

// Summary 展示串（CLI 表格 / alfred 副标题 / Web DTO）：
// 按 role 固定序拼接 "role:动作串"。
func (o Opener) Summary() string {
	parts := make([]string, 0, len(o.Actions))
	for _, r := range o.Roles() {
		parts = append(parts, string(r)+":"+o.Actions[r])
	}
	return strings.Join(parts, "; ")
}
