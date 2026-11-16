package handlers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/api"
	"cube/internal/domain/opener"
)

// OpenerHandler opener 域 HTTP 出口。
type OpenerHandler struct {
	openerService *opener.Service
}

// NewOpenerHandler 构造 OpenerHandler。
func NewOpenerHandler(openerService *opener.Service) *OpenerHandler {
	return &OpenerHandler{
		openerService: openerService,
	}
}

// Register 注册 opener 域端点（实现 api.Registerer）。
func (h *OpenerHandler) Register(r *api.Routes) {
	r.Get("/api/opener/list", "获取 opener 列表", api.JSONHandler(h.list))
	r.Get("/api/opener/intents", "获取打开意图清单（intent → 默认 opener + 候选）", api.JSONHandler(h.intents))
	r.Post("/api/opener/open", "用指定 opener 打开任意文件或目录", api.JSONHandler(h.openPath))
	r.Post("/api/opener/diff-open", "用指定 opener 对比两个路径（diff-dir/diff-file）", api.JSONHandler(h.openDiff))
	r.Post("/api/opener/save", "新增或更新 opener（按 name 替换）", api.JSONHandler(h.save))
	r.Post("/api/opener/delete", "删除 opener（提交完整单项，按 name 定位）", api.JSONHandler(h.delete))
	r.Post("/api/opener/save-all", "整节覆盖 opener（提交即落盘，含顺序）", api.JSONHandler(h.saveAll))
	r.Post("/api/opener/intent-default/save", "设置某 intent 的默认 opener", api.JSONHandler(h.intentDefaultSave))
	r.Post("/api/opener/intent-default/delete", "清除某 intent 的默认 opener", api.JSONHandler(h.intentDefaultDelete))
}

/**
 * GET /api/opener/list
 */

func (h *OpenerHandler) list(_ struct{}) (*ListResult[OpenerDTO], error) {
	openers := h.openerService.AllOpeners()
	list := xslices.Map(openers, toOpenerDTO)
	return listResult(list), nil
}

/**
 * GET /api/opener/intents
 */

func (h *OpenerHandler) intents(_ struct{}) (*ListResult[OpenerIntentDTO], error) {
	intentInfos := h.openerService.Intents()
	list := xslices.Map(intentInfos, toOpenerIntentDTO)
	return listResult(list), nil
}

/**
 * POST /api/opener/open
 */

// openPath 用指定 opener 打开任意文件/目录（项目打开走 project/open，那里记 usage；此处不产生使用信号）。
// role 按路径实际类型校验：目录须 open-dir、文件须 open-file。
func (h *OpenerHandler) openPath(input WithBody[struct {
	Path   string `json:"path" doc:"文件或目录绝对路径"`
	Opener string `json:"opener" doc:"opener 名称"`
}]) (*OpResult, error) {
	if !filepath.IsAbs(input.Body.Path) {
		return nil, errors.New("path 必须是绝对路径: " + input.Body.Path)
	}

	info, err := os.Stat(input.Body.Path)
	if err != nil {
		return nil, fmt.Errorf("读取路径失败: %w", err)
	}

	o := h.openerService.FindByName(input.Body.Opener)
	if o == nil {
		return nil, fmt.Errorf("未找到指定 opener: %s", input.Body.Opener)
	}

	role := opener.RoleOpenFile
	if info.IsDir() {
		role = opener.RoleOpenDir
	}

	// role 是否支持由 Service.Open 首步校验，这里只透传错误
	if err := h.openerService.Open(*o, role, input.Body.Path); err != nil {
		return nil, fmt.Errorf("打开失败: %w", err)
	}
	return opSuccess, nil
}

/**
 * POST /api/opener/diff-open
 */

// openDiff 用指定 opener 对比两个路径。role 按两侧实际类型推导且须同类
// （两侧都是目录 → diff-dir，都是文件 → diff-file，混合报错）。
func (h *OpenerHandler) openDiff(input WithBody[struct {
	Left   string `json:"left" doc:"左侧路径（文件或目录）绝对路径"`
	Right  string `json:"right" doc:"右侧路径（文件或目录）绝对路径"`
	Opener string `json:"opener" doc:"opener 名称"`
}]) (*OpResult, error) {
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

	o := h.openerService.FindByName(input.Body.Opener)
	if o == nil {
		return nil, fmt.Errorf("未找到指定 opener: %s", input.Body.Opener)
	}

	role := opener.RoleDiffFile
	if leftDir {
		role = opener.RoleDiffDir
	}

	if err := h.openerService.Open(*o, role, left, right); err != nil {
		return nil, fmt.Errorf("打开失败: %w", err)
	}
	return opSuccess, nil
}

func isDirPath(p string) (bool, error) {
	info, err := os.Stat(p)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

/**
 * POST /api/opener/save
 */

func (h *OpenerHandler) save(input WithBody[OpenerItem]) (*OpResult, error) {
	// 校验在 Service 写侧（领域构造函数），坏数据返回中文错误、不落文件
	if err := h.openerService.SaveOpener(input.Body.toSpec()); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/opener/delete
 */

func (h *OpenerHandler) delete(input WithBody[OpenerItem]) (*OpResult, error) {
	if err := h.openerService.DeleteOpener(input.Body.toSpec()); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/opener/save-all
 */

func (h *OpenerHandler) saveAll(input WithBody[struct {
	Items []OpenerItem `json:"items" doc:"按目标顺序排列的全部 opener"`
}]) (*OpResult, error) {
	if err := h.openerService.SaveAllOpeners(xslices.Map(input.Body.Items, OpenerItem.toSpec)); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/opener/intent-default/save
 */

// intent-default/save 接口入参。
func (h *OpenerHandler) intentDefaultSave(input WithBody[struct {
	Intent string `json:"intent" doc:"打开意图（如 dir / file / diff-file / git）"`
	Opener string `json:"opener" doc:"opener 名称（须声明该 intent 对应的 role）"`
}]) (*OpResult, error) {
	intent, err := opener.ParseIntent(input.Body.Intent)
	if err != nil {
		return nil, err
	}
	if err := h.openerService.SaveIntentDefault(intent, input.Body.Opener); err != nil {
		return nil, err
	}
	return opSuccess, nil
}

/**
 * POST /api/opener/intent-default/delete
 */

// intent-default/delete 接口入参。
func (h *OpenerHandler) intentDefaultDelete(input WithBody[struct {
	Intent string `json:"intent" doc:"打开意图"`
}]) (*OpResult, error) {
	intent, err := opener.ParseIntent(input.Body.Intent)
	if err != nil {
		return nil, err
	}
	if err := h.openerService.DeleteIntentDefault(intent); err != nil {
		return nil, err
	}
	return opSuccess, nil
}
