// Package logger 日志初始化：Init 统一构造文件/stdio handler 并 slog.SetDefault。
// 业务代码直接用 slog 打日志（slog.Info 等），不允许使用标准库 log 包。
//
// 日志策略（CLI 工具形态）：JSON 文件日志始终启用，作为权威记录；
// stdio 着色输出仅在 Debug 模式附加——平时保持终端干净，排查时才双写。
// 未 Init 时走 slog 内置默认输出（stderr、Info 级）。
package logger
