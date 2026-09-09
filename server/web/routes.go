package web

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"cube/version"
)

// Routes 路由注册器：持有 huma API 与底层 http mux，二者不对 handler 层暴露。
//
// handler 通过 Register(r *Routes) 注册路由：标准查询/动作走 Get / Post
// （输入/输出类型经 JsonHandler / RawHandler 包装携带，见 TypedHandler），
// 进 OpenAPI 文档，个别端点可用 WithoutOpenAPI 显式排除；需要原始控制
// ResponseWriter 的端点（连接升级 / 二进制流）走 Raw，一律不进文档。
// 换路由实现（如 chi）时只改本文件，handlers 包不感知。
type Routes struct {
	api huma.API
	mux *http.ServeMux
}

// newRoutes 构造注册器，同时完成 huma API 与 OpenAPI 文档的基础配置。
func newRoutes() *Routes {
	mux := http.NewServeMux()

	cfg := huma.DefaultConfig(version.AppTitle, version.Version())
	cfg.DocsRenderer = huma.DocsRendererScalar // 切换 /docs 页面风格为 Scalar 渲染器
	cfg.Formats = map[string]huma.Format{
		"application/json": nilCollectionsJSONFormat, // nil 切片/map → []/{}，避免前端拿到 null 崩溃
		"json":             nilCollectionsJSONFormat, // "json" 别名必须存在：错误响应的 application/problem+json 靠取 "+" 后的 "json" 兜底查表
	}
	api := humago.New(mux, cfg)

	return &Routes{api: api, mux: mux}
}

// --- 查询接口 ---

// Handler 返回底层 http.Handler，供 http.Server 挂载与 httptest 拉起真实路由做集成测试。
func (r *Routes) Handler() http.Handler { return r.mux }

// OpenAPIJSON 返回 OpenAPI 3.1 spec 的 JSON 字节。供 generate 命令或 /openapi.json 端点使用。
func (r *Routes) OpenAPIJSON() ([]byte, error) {
	return r.api.OpenAPI().MarshalJSON()
}

// --- 类型化路由（Get / Post）---

// Get 注册 GET 查询路由
func (r *Routes) Get(path string, summary string, h TypedHandler, opts ...RouteOption) {
	op := prepareOperation(huma.Operation{
		Method:  http.MethodGet,
		Path:    path,
		Summary: summary,
	}, opts...)
	h.registerOn(r, op)
}

// Post 注册 POST 动作路由
func (r *Routes) Post(path string, summary string, h TypedHandler, opts ...RouteOption) {
	op := prepareOperation(huma.Operation{
		Method:  http.MethodPost,
		Path:    path,
		Summary: summary,
	}, opts...)
	h.registerOn(r, op)
}

// --- 原生路由（Raw）---

// Raw 原样透传给底层 mux 挂载，不经 huma、不进 OpenAPI 文档。
// pattern 为完整 method+path（如 "GET /api/workbench/pty"）。
func (r *Routes) Raw(pattern string, h http.Handler) {
	r.mux.Handle(pattern, h)
}

// TypedHandler 及其使用

// TypedHandler 携带输入/输出类型信息的 huma handler——Go 泛型不能定义在方法上，
// 由包装函数（JsonHandler / RawHandler）把类型参数捕获进闭包，供 Get/Post 的
// 非泛型方法触发注册；包装函数同时决定响应形态（envelope 与否）。
type TypedHandler interface {
	registerOn(r *Routes, op huma.Operation)
}

type typedRoute[I, O any] struct {
	h func(context.Context, *I) (*O, error)
}

func (t typedRoute[I, O]) registerOn(r *Routes, op huma.Operation) {
	huma.Register(r.api, op, t.h)
}

// RawHandler 把 huma 签名的 handler 装进非泛型载体。
func RawHandler[I, O any](h func(context.Context, *I) (*O, error)) TypedHandler {
	return typedRoute[I, O]{h: h}
}

// JsonOutput api 标准输出结构
type JsonOutput[T any] struct {
	Body struct {
		Ok      bool   `json:"ok"`
		Message string `json:"message"`
		Data    *T     `json:"data"`
	}
}

// JsonHandler 包装 handler 并包 JsonOutput envelope——标准 JSON 端点的默认形态：
// r.Get("/api/foo/list", "查询列表", web.Json(h.list))
func JsonHandler[I, O any](h func(I) (O, error)) TypedHandler {
	return RawHandler(func(_ context.Context, input *I) (*JsonOutput[O], error) {
		data, err := h(*input)

		var output JsonOutput[O]
		if err != nil {
			output.Body.Ok = false
			output.Body.Message = err.Error()
			output.Body.Data = nil
		} else {
			output.Body.Ok = true
			output.Body.Message = ""
			output.Body.Data = &data
		}
		return &output, nil
	})
}

// --- 通用选项与 operation 默认值 ---

// RouteOption 路由注册可选项，Get / Post / Raw 通用，直接作用于 huma.Operation。
type RouteOption func(*huma.Operation)

// WithoutOpenAPI 声明该路由不进 OpenAPI 文档——用于 OpenAPI 无法表达的
// 端点（WebSocket 连接升级）或刻意不暴露的运维端点。路由功能不受影响。
func WithoutOpenAPI() RouteOption {
	return func(op *huma.Operation) { op.Hidden = true }
}

// WithSummary 覆盖默认 summary。
func WithSummary(summary string) RouteOption {
	return func(op *huma.Operation) { op.Summary = summary }
}

// prepareOperation 应用用户选项后补默认值（用户显式设置优先，不被覆盖），
// 并校验 /api/ 前缀——所有 API 路由必须落在 /api/ 下，其余路径留给静态资源。
func prepareOperation(op huma.Operation, opts ...RouteOption) huma.Operation {
	applyRouteOptions(&op, opts...)

	if op.Method == "" {
		panic("operation 必须指定 method")
	}
	if op.Path == "" {
		panic("operation 必须指定 path")
	} else if !strings.HasPrefix(op.Path, "/api/") {
		panic("api path 必须以 /api/ 开头: " + op.Path)
	}

	group, operationId := parseInfoFromPath(op.Path)

	// 默认将分组作为 tag
	if !slices.Contains(op.Tags, group) {
		op.Tags = slices.Concat(op.Tags, []string{group})
	}

	// 默认 operation-id，影响使用方的代码生成
	if op.OperationID == "" {
		op.OperationID = operationId
	}

	return op
}

func applyRouteOptions(op *huma.Operation, opts ...RouteOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func parseInfoFromPath(p string) (group string, operationId string) {
	p = strings.Trim(strings.TrimPrefix(p, "/api"), " /")
	if p == "" {
		return "index", "index.index"
	}

	if idx := strings.IndexByte(p, '/'); idx < 0 {
		return p, p + ".index"
	} else {
		group = p[:idx]
		rest := toCamelPath(p[idx+1:])
		operationId = group + "." + rest
		return
	}
}

// toCamelPath 把路径段转驼峰："/clone-rules" → "cloneRules"（"-"后字母大写，其余原样）
func toCamelPath(p string) string {
	var b strings.Builder
	upper := false
	for _, r := range p {
		switch {
		case r == '-' || r == '/':
			upper = true
		case upper:
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
