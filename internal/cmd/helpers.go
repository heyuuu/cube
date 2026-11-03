package cmd

import (
	"errors"
	"fmt"

	"cube/internal/domain/opener"
	"cube/pkg/tui"
)

// interactivePick `-o` 不带值时的哨兵（flag NoOptDefVal）：触发交互选择。
const interactivePick = "?"

// pickOpener 按打开意图挑选 opener（1038）：
//   - name 为空（未传 -o）→ 直接用该 intent 的默认 opener，未配置时报错引导；
//   - name 为交互哨兵 "?"（-o 不带值）→ 在该 intent 对应 role 的候选中模糊匹配，
//     命中多个且在 TTY 环境时交互选择；
//   - name 为具体值 → 按名模糊匹配（现有语义不变）。
//
// 非交互环境不支持多项选择，会报错提示使用精确 opener 名。
func pickOpener(service *opener.Service, intent opener.Intent, name string) (opener.Opener, error) {
	if name == "" {
		return service.DefaultOpener(intent)
	}
	if name == interactivePick {
		name = ""
	}
	role := intent.Role()
	openers := service.SearchFor(role, name)
	if len(openers) == 0 {
		return nil, fmt.Errorf("未找到匹配的 opener: intent=%s, name=`%s`", intent, name)
	} else if len(openers) == 1 {
		return openers[0], nil
	}

	// 匹配多个 opener 时，触发用户选择
	pick, err := tui.SelectItem("选择 opener", openers, opener.Opener.Name)
	if err != nil {
		if errors.Is(err, tui.ErrNotTTY) {
			return nil, fmt.Errorf("非 TTY 环境请使用精确 opener 名，避免匹配多项: name=`%s`", name)
		}
		if errors.Is(err, tui.ErrUserAborted) {
			return nil, fmt.Errorf("用户取消了 opener 选择: %w", err)
		}
		return nil, err
	}
	return pick, nil
}
