package handlers

import (
	"cube/internal/domain/template"
)

// TplSourceItem 模板源单项（save / delete / save-all 共用，字段与 template.TplSource 对齐）。
type TplSourceItem struct {
	Name    string `json:"name" doc:"source 名（cube create 的引用短名，如 core）"`
	RepoURL string `json:"repoUrl" doc:"git 仓库地址（git@host:path / https://host/path / 本地绝对路径）"`
}

func (it TplSourceItem) toTplSource() template.TplSource {
	return template.TplSource{Name: it.Name, RepoURL: it.RepoURL}
}
