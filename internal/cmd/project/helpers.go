package project

import (
	"cube/internal/cmd/cmdutil"
	"cube/internal/domain/project"
)

// pickProject cmdutil.PickProject 的别名
// pickProject 理论上是 cmd/project 包内函数，但是放在了 cmd/cmdutil 方便共享；此处留一个别名在包内使用
func pickProject(service *project.Service, query string, localMode bool) (*project.Project, error) {
	return cmdutil.PickProject(service, query, localMode)
}
