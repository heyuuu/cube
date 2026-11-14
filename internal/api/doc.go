// Package api Web API 契约层：Routes 路由注册、Envelope 输出信封、JSONHandler
// 包装与 shutdown 鉴权 token。业务 handler（internal/handlers）与运行时
// （internal/server）都依赖本包，二者互不感知。
package api
