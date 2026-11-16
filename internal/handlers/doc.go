// Package handlers 是业务 HTTP handler 层：把领域 service 适配为 HTTP API。
//
// 组织方式：同包按 domain 分文件（<domain>_handler.go），不按 domain 分子包——
// handler 之间无共享状态，横向扩展只加文件。路由注册与 envelope 契约在
// internal/api（注册统一走 r.Get() / r.Post()），server 运行时（监听/优雅关闭）
// 在 internal/server
package handlers
