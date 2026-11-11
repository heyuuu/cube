package opener

import (
	"fmt"

	"cube/internal/domain/icons"
)

// Spec opener 在 settings.json openers 节内的存储形状。
// 节内结构是 opener 域的知识，DTO 归本包（settings 包不感知节内形状）。
type Spec struct {
	Name    string          `json:"name"`
	Title   string          `json:"title,omitempty"` // 展示文案（如「打开所在目录」「使用 VS Code 打开」），缺省由 name 生成
	Actions map[Role]string `json:"actions"`         // role → 动作串 `<kind>:<模板>`（kind: exec 命令 / url 链接，见 exec.go）；$0/$1... 占位路径槽位，可嵌在模板内（如 --wd=$0）。声明了哪些 role 即键集合
	Icon    *icons.Icon     `json:"icon,omitempty"`  // 图标声明（前端渲染方式），缺省无图标
}

// specName opener 在 openers 节内的唯一键（name）。
func specName(s Spec) string { return s.Name }

func validateSpec(spec Spec) (Spec, error) {
	var zero Spec
	if spec.Name == "" {
		return zero, fmt.Errorf("opener name 不得为空")
	}
	if len(spec.Actions) == 0 {
		return zero, fmt.Errorf("opener %q 缺少必填字段 actions", spec.Name)
	}

	if spec.Title == "" {
		spec.Title = "用 " + spec.Name + " 打开"
	}

	icon, err := InitIcon(spec.Icon)
	if err != nil {
		return zero, fmt.Errorf("opener %q icon 解析失败: %w", spec.Name, err)
	}
	spec.Icon = &icon

	for role, raw := range spec.Actions {
		if _, err := validateAction(role, raw); err != nil {
			return zero, err
		}
	}

	return spec, nil
}
