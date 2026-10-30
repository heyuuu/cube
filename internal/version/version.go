// Package version 暴露构建期信息。默认值用于 dev；正式构建用 ldflags 注入。
// e.g. go build -ldflags "-X 'cube/internal/version.version=v3.0.0'"
package version

import (
	"fmt"
)

var (
	version   = "dev" // -ldflags 注入，默认 dev 标记
	commit    = "unknown"
	buildTime = "unknown"
)

// IsDev 判断当前是否在开发环境
func IsDev() bool {
	return version == "dev"
}

// Version 返回版本号；未经 ldflags 注入时为 "dev"。
func Version() string {
	return version
}

// Info 返回一行可读的完整版本信息：dev 只含版本号，正式构建附 commit 与构建时间。
func Info() string {
	if IsDev() {
		return version
	}
	return fmt.Sprintf("%s (%s %s)", version, commit, buildTime)
}

// AppName 应用标识（小写无空格）：配置目录名 / status 身份 / 进程探测 / 日志文件名都由它派生。
// 模板项目初始化时只需改这里 + 构建工具链的替换表。
const AppName = "cube"
