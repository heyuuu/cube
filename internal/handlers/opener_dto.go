package handlers

import (
	"cube/internal/domain/icons"
	"cube/internal/domain/opener"
)

// OpenerDTO opener 列表 / 编辑回显的 API 形态。
type OpenerDTO struct {
	Name    string            `json:"name"`
	Title   string            `json:"title"`   // 展示文案（如「打开所在目录」），缺省由 name 生成
	Summary string            `json:"summary"` // 动作串展示串（"role:动作串" 拼接）
	Actions map[string]string `json:"actions"` // role → 动作串原文（`<kind>:<模板>`，编辑表单回显用）；键集合即声明的 roles
	Icon    *icons.Icon       `json:"icon"`    // 恒有值（未配置时后端按主 role 填默认 lucide 图）
}

func toOpenerDTO(entity opener.Opener) OpenerDTO {
	actions := make(map[string]string, len(entity.Actions))
	for r, action := range entity.Actions {
		actions[string(r)] = action
	}
	return OpenerDTO{
		Name:    entity.Name,
		Title:   entity.Title,
		Summary: entity.Summary(),
		Actions: actions,
		Icon:    entity.Icon,
	}
}

// OpenerIntentDTO intents 接口输出条目。
type OpenerIntentDTO struct {
	Intent        string   `json:"intent"`
	DefaultOpener string   `json:"defaultOpener,omitempty"`
	Openers       []string `json:"openers"` // 候选清单（缺省 = 声明了对应 role 的全部 opener，读侧合成）
}

func toOpenerIntentDTO(i opener.IntentInfo) OpenerIntentDTO {
	return OpenerIntentDTO{
		Intent:        string(i.Intent),
		DefaultOpener: i.DefaultOpener,
		Openers:       i.Openers,
	}
}

// OpenerItem opener 单项（save / delete / save-all 共用，字段与 opener.Opener 对齐）。
type OpenerItem struct {
	Name    string            `json:"name" doc:"opener 名称（唯一标识）"`
	Title   string            `json:"title,omitempty" doc:"展示文案（如「打开所在目录」），缺省由 name 生成"`
	Actions map[string]string `json:"actions" doc:"role → 动作串，exec: 命令 / url: 链接，$0/$1 占位路径槽位"`
	Icon    *icons.Icon       `json:"icon,omitempty" doc:"图标声明"`
}

func (it OpenerItem) toSpec() opener.Opener {
	actions := make(map[opener.Role]string, len(it.Actions))
	for r, raw := range it.Actions {
		actions[opener.Role(r)] = raw
	}
	return opener.Opener{Name: it.Name, Title: it.Title, Actions: actions, Icon: it.Icon}
}
