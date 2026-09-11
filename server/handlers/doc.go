// Package handlers 是业务 HTTP handler 层：把领域 service 适配为 HTTP API。
//
// 组织方式：同包按 domain 分文件（<domain>_handler.go），不按 domain 分子包——
// handler 之间无共享状态，横向扩展只加文件。服务端框架（路由 / envelope /
// 静态资源 / system 端点）在 cube/core/server，注册统一走 Register(r *server.Routes)：
// 标准端点 r.Get / r.Post + server.JsonHandler（envelope）或 server.RawHandler（透传），
// 需要原始控制 ResponseWriter 的端点（WebSocket / 二进制流）走 r.Raw。
package handlers
