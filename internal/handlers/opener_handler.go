package handlers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/domain/icons"
	"cube/internal/domain/opener"
	"cube/internal/server"
)

// --- dto ---

// OpenerDTO opener 列表 / 编辑回显的 API 形态。
type OpenerDTO struct {
	Name    string            `json:"name"`
	Title   string            `json:"title"`   // 展示文案（如「打开所在目录」），缺省由 name 生成
	Summary string            `json:"summary"` // 动作串展示串（"role:动作串" 拼接）
	Actions map[string]string `json:"actions"` // role → 动作串原文（`<kind>:<模板>`，编辑表单回显用）；键集合即声明的 roles
	Icon    icons.Icon        `json:"icon"`    // 恒有值（未配置时后端按主 role 填默认 lucide 图）
}

func toOpenerDTO(entity opener.Opener) *OpenerDTO {
	if entity == nil {
		return nil
	}

	actions := entity.Actions()
	dtoActions := make(map[string]string, len(actions))
	for r, raw := range actions {
		dtoActions[string(r)] = raw
	}
	return &OpenerDTO{
		Name:    entity.Name(),
		Title:   entity.Title(),
		Summary: entity.Summary(),
		Actions: dtoActions,
		Icon:    entity.Icon(),
	}
}

// --- handler ---

// OpenerHandler opener 域 HTTP 出口。
type OpenerHandler struct {
	service *opener.Service
}

// NewOpenerHandler 构造 OpenerHandler。
func NewOpenerHandler(service *opener.Service) *OpenerHandler {
	return &OpenerHandler{
		service: service,
	}
}

// Register 注册 opener 域端点（实现 server.Handler）。
func (h *OpenerHandler) Register(r *server.Routes) {
	r.Get("/api/opener/list", "获取 opener 列表", server.JSONHandler(h.openerList))
	r.Get("/api/opener/intents", "获取打开意图清单（intent → 默认 opener + 候选）", server.JSONHandler(h.openerIntents))
	r.Post("/api/opener/open", "用指定 opener 打开任意文件或目录", server.JSONHandler(h.openerOpen))
	r.Post("/api/opener/diff-open", "用指定 opener 对比两个路径（diff-dir/diff-file）", server.JSONHandler(h.openerDiffOpen))
	r.Post("/api/opener/save", "新增或更新 opener（按 name 替换）", server.JSONHandler(h.openerSave))
	r.Post("/api/opener/delete", "删除 opener（提交完整单项，按 name 定位）", server.JSONHandler(h.openerDelete))
	r.Post("/api/opener/save-all", "整节覆盖 opener（提交即落盘，含顺序）", server.JSONHandler(h.openerSaveAll))
	r.Post("/api/opener/intent-default/save", "设置某 intent 的默认 opener", server.JSONHandler(h.intentDefaultSave))
	r.Post("/api/opener/intent-default/delete", "清除某 intent 的默认 opener", server.JSONHandler(h.intentDefaultDelete))
}

func (h *OpenerHandler) openerList(_ struct{}) (ListResult[*OpenerDTO], error) {
	openers := h.service.AllOpeners()
	list := xslices.Map(openers, toOpenerDTO)
	return listResult(list), nil
}

// OpenerIntentDTO intents 接口输出条目。
type OpenerIntentDTO struct {
	Intent        string   `json:"intent"`
	DefaultOpener string   `json:"defaultOpener,omitempty"`
	Openers       []string `json:"openers"` // 候选清单（缺省 = 声明了对应 role 的全部 opener，读侧合成）
}

func (h *OpenerHandler) openerIntents(_ struct{}) (ListResult[*OpenerIntentDTO], error) {
	list := xslices.Map(h.service.Intents(), func(i opener.IntentInfo) *OpenerIntentDTO {
		return &OpenerIntentDTO{
			Intent:        string(i.Intent),
			DefaultOpener: i.DefaultOpener,
			Openers:       i.Openers,
		}
	})
	return listResult(list), nil
}

// OpenerOpenInput open 接口入参。huma 约定：请求体字段须挂在名为 Body 的子结构上。
type OpenerOpenInput struct {
	Body struct {
		Path   string `json:"path" doc:"文件或目录绝对路径"`
		Opener string `json:"opener" doc:"opener 名称"`
	}
}

// openerOpen 用指定 opener 打开任意文件/目录（项目打开走 project/open，那里记 usage；此处不产生使用信号）。
// role 按路径实际类型校验：目录须 open-dir、文件须 open-file。
func (h *OpenerHandler) openerOpen(input OpenerOpenInput) (map[string]any, error) {
	if !filepath.IsAbs(input.Body.Path) {
		return nil, errors.New("path 必须是绝对路径: " + input.Body.Path)
	}
	info, err := os.Stat(input.Body.Path)
	if err != nil {
		return nil, fmt.Errorf("读取路径失败: %w", err)
	}

	o := h.service.FindByName(input.Body.Opener)
	if o == nil {
		return nil, fmt.Errorf("未找到指定 opener: %s", input.Body.Opener)
	}
	role := opener.RoleOpenFile
	if info.IsDir() {
		role = opener.RoleOpenDir
	}

	// role 是否支持由实现内校验（Open 首步），这里只透传错误
	if err := o.Open(role, input.Body.Path); err != nil {
		return nil, fmt.Errorf("打开失败: %w", err)
	}
	return map[string]any{"ok": true}, nil
}

// OpenerDiffOpenInput diff-open 接口入参。
type OpenerDiffOpenInput struct {
	Body struct {
		Left   string `json:"left" doc:"左侧路径（文件或目录）绝对路径"`
		Right  string `json:"right" doc:"右侧路径（文件或目录）绝对路径"`
		Opener string `json:"opener" doc:"opener 名称"`
	}
}

// openerDiffOpen 用指定 opener 对比两个路径。role 按两侧实际类型推导且须同类
// （两侧都是目录 → diff-dir，都是文件 → diff-file，混合报错）。
func (h *OpenerHandler) openerDiffOpen(input OpenerDiffOpenInput) (map[string]any, error) {
	left, right := input.Body.Left, input.Body.Right
	for _, p := range []string{left, right} {
		if !filepath.IsAbs(p) {
			return nil, errors.New("路径必须是绝对路径: " + p)
		}
	}
	leftDir, err := isDirPath(left)
	if err != nil {
		return nil, fmt.Errorf("读取路径失败: %w", err)
	}
	rightDir, err := isDirPath(right)
	if err != nil {
		return nil, fmt.Errorf("读取路径失败: %w", err)
	}
	if leftDir != rightDir {
		return nil, errors.New("两侧路径类型不一致（一边是目录一边是文件），无法对比")
	}

	o := h.service.FindByName(input.Body.Opener)
	if o == nil {
		return nil, fmt.Errorf("未找到指定 opener: %s", input.Body.Opener)
	}
	role := opener.RoleDiffFile
	if leftDir {
		role = opener.RoleDiffDir
	}
	if err := o.Open(role, left, right); err != nil {
		return nil, fmt.Errorf("打开失败: %w", err)
	}
	return map[string]any{"ok": true}, nil
}

func isDirPath(p string) (bool, error) {
	info, err := os.Stat(p)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// OpenerItem opener 单项（save / delete / save-all 共用，字段与 opener.Spec 对齐）。
type OpenerItem struct {
	Name    string            `json:"name" doc:"opener 名称（唯一标识）"`
	Title   string            `json:"title,omitempty" doc:"展示文案（如「打开所在目录」），缺省由 name 生成"`
	Actions map[string]string `json:"actions" doc:"role → 动作串，exec: 命令 / url: 链接，$0/$1 占位路径槽位"`
	Icon    *icons.Icon       `json:"icon,omitempty" doc:"图标声明"`
}

func (it OpenerItem) toSpec() opener.Spec {
	actions := make(map[opener.Role]string, len(it.Actions))
	for r, raw := range it.Actions {
		actions[opener.Role(r)] = raw
	}
	return opener.Spec{Name: it.Name, Title: it.Title, Actions: actions, Icon: it.Icon}
}

// OpenerSaveInput save 接口入参。
type OpenerSaveInput struct {
	Body OpenerItem
}

func (h *OpenerHandler) openerSave(input OpenerSaveInput) (map[string]any, error) {
	// 校验在 Service 写侧（领域构造函数），坏数据返回中文错误、不落文件
	if err := h.service.SaveOpener(input.Body.toSpec()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// OpenerDeleteInput delete 接口入参（完整单项，唯一键由 domain 计算）。
type OpenerDeleteInput struct {
	Body OpenerItem
}

func (h *OpenerHandler) openerDelete(input OpenerDeleteInput) (map[string]any, error) {
	if err := h.service.DeleteOpener(input.Body.toSpec()); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// OpenerSaveAllInput save-all 接口入参（提交清单即落盘结果，含顺序）。
type OpenerSaveAllInput struct {
	Body struct {
		Items []OpenerItem `json:"items" doc:"按目标顺序排列的全部 opener"`
	}
}

func (h *OpenerHandler) openerSaveAll(input OpenerSaveAllInput) (map[string]any, error) {
	if err := h.service.SaveAllOpeners(xslices.Map(input.Body.Items, OpenerItem.toSpec)); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// IntentDefaultSaveInput intent-default/save 接口入参。
type IntentDefaultSaveInput struct {
	Body struct {
		Intent string `json:"intent" doc:"打开意图（如 dir / file / diff-file / git）"`
		Opener string `json:"opener" doc:"opener 名称（须声明该 intent 对应的 role）"`
	}
}

func (h *OpenerHandler) intentDefaultSave(input IntentDefaultSaveInput) (map[string]any, error) {
	intent, err := opener.ParseIntent(input.Body.Intent)
	if err != nil {
		return nil, err
	}
	if err := h.service.SaveIntentDefault(intent, input.Body.Opener); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// IntentDefaultDeleteInput intent-default/delete 接口入参。
type IntentDefaultDeleteInput struct {
	Body struct {
		Intent string `json:"intent" doc:"打开意图"`
	}
}

func (h *OpenerHandler) intentDefaultDelete(input IntentDefaultDeleteInput) (map[string]any, error) {
	intent, err := opener.ParseIntent(input.Body.Intent)
	if err != nil {
		return nil, err
	}
	if err := h.service.DeleteIntentDefault(intent); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}
