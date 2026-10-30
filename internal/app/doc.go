// Package app 进程级上下文与组装层：App 先建空壳供 cmd 构建期持有，
// Init 完成配置加载与 DI 容器组装，server 与各领域服务统一从这里取用。
package app
