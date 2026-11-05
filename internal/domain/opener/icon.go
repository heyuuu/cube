package opener

import (
	"cube/internal/domain/icons"
)

// icon 类型枚举（透传 icons domain）。
const (
	IconTypeLucide = icons.IconTypeLucide
	IconTypeImage  = icons.IconTypeImage
)

// InitIcon 校验 icon 声明并解析默认值。nil 视为未配置（合法），返回默认 icon——
// Opener 对外永远有 Icon，调用方无需处理缺省；type 非法或 value 为空报中文错误。
func InitIcon(icon *icons.Icon) (icons.Icon, error) {
	if icon == nil {
		return defaultIcon(), nil
	}
	if err := icons.ValidateIcon(icon); err != nil {
		return icons.Icon{}, err
	}
	return *icon, nil
}

// defaultIcon 未配置 icon 时按主 role 推导的 lucide 默认图（前端按名渲染）。
func defaultIcon() icons.Icon {
	return icons.Icon{Type: IconTypeLucide, Value: "app-window-mac"}
}
