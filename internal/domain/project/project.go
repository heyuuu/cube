// Package project 项目领域：从扫描规则识别项目（git 仓库），提供搜索排序、
// clone 规则匹配与打开目标（worktree / workspace）枚举。
package project

import "path/filepath"

// Project 扫描识别出的一个项目（数据由 scan-rule 推导，只读快照）。
type Project struct {
	path  string   // 项目路径，唯一标识
	group string   // 所属工作区名
	name  string   // 项目展示名，格式 `{组名}:{组内相对路径}`
	tags  []string // 标签列表
}

func newProject(r ScanRule, path string, tags []string) *Project {
	// 构建项目数据
	return &Project{
		path:  path,
		group: r.Group,
		name:  projectName(r, path),
		tags:  tags,
	}
}

// projectName 计算项目展示名 `{组名}:{组内相对路径}`。
// 若整个工作区即为当前项目（path 即规则根），则直接使用工作区名。
func projectName(r ScanRule, path string) string {
	subName, _ := filepath.Rel(r.Path, path)
	if subName == "." {
		subName = r.Group
	}
	return r.Group + ":" + subName
}

// Path 返回项目绝对路径（唯一标识）。
func (p Project) Path() string { return p.path }

// Group 返回所属工作区名。
func (p Project) Group() string { return p.group }

// Name 返回展示名（{组名}:{组内相对路径}）。
func (p Project) Name() string { return p.name }

// Tags 返回特征 tag 列表。
func (p Project) Tags() []string { return p.tags }
