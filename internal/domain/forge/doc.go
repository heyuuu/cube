// Package forge 管理 git 托管平台实例（forge）配置。
// forge 是 host 级的一条配置（如 github.com、自建 gitea.example.com），
// 承载三件事：host → icon 展示、kind 决定 API 方言（1041 account 拉取据此分发）、
// 作为后续 account（1041）与 forge 页（1042）的挂靠点。
//
// 限定 git 仓库的托管平台（项目前提是所有项目都是 git 项目，repo→forge 匹配
// 走 git remote URL 的 host 解析）；数据存 settings.json 的 "forges" 节。
package forge
