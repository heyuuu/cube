package icons

import "fmt"

// IconType icon 声明的渲染类型。
type IconType string

// icon 类型枚举：lucide=按图名渲染 lucide-react，image=base64 PNG 直渲染。
const (
	IconTypeImage  IconType = "image"
	IconTypeLucide IconType = "lucide"
)

// Icon 图标声明：只判别「前端怎么渲染」，不管图片来源。
//   - type=lucide：前端按 value 的图名渲染 lucide-react 图标；
//   - type=image：value 是 base64 PNG，前端直接渲染（.app 提取 / 上传图片两个入口
//     都先落成 base64，见本包 ExtractAppIcon）。
//
// 指针 nil 表示未配置（可选字段）；未配置时的兜底策略（显示默认图 / 显示占位）
// 归各领域自行决定，本包不预设默认值。
type Icon struct {
	Type  IconType `json:"type"`
	Value string   `json:"value"`
}

// ValidateIcon 校验 icon 声明。nil 视为未配置（合法，可选语义）；
// type 非法或 value 为空报中文错误。
func ValidateIcon(icon *Icon) error {
	if icon == nil {
		return nil
	}
	switch icon.Type {
	case IconTypeLucide, IconTypeImage:
	default:
		return fmt.Errorf("未知的 icon type %q（合法值：lucide/image）", icon.Type)
	}
	if icon.Value == "" {
		return fmt.Errorf("icon value 不得为空（type=%s）", icon.Type)
	}
	return nil
}
