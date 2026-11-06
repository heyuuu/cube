// Package forge 管理 git 托管平台实例（forge）配置。
// forge 是 host 级的一条配置（如 github.com、自建 gitea.example.com），
// 承载三件事：host → icon 展示、kind 决定 API 方言（1041 account 拉取据此分发）、
// 作为后续 account（1041）与 forge 页（1042）的挂靠点。
//
// 限定 git 仓库的托管平台（项目前提是所有项目都是 git 项目，repo→forge 匹配
// 走 git remote URL 的 host 解析）；数据存 settings.json 的 "forges" 节。
package forge

import (
	"fmt"
	"strings"

	"cube/internal/domain/icons"
	"cube/pkg/git"
)

// Forge 一条 forge 配置，host 是唯一键。
type Forge struct {
	Host string      `json:"host"`           // 域名（可带端口），如 github.com、gitea.example.com:3000
	Kind string      `json:"kind"`           // API 方言，见 Kind* 常量
	Icon *icons.Icon `json:"icon,omitempty"` // 可选图标（前端按此渲染）
}

// kind 枚举：决定 1041 account 能否拉取及 API 方言；generic = 无 API 仅展示。
const (
	KindGithub  = "github"
	KindGitea   = "gitea"
	KindGitee   = "gitee"
	KindGeneric = "generic"
)

// Kinds 全部合法 kind 值（展示与校验共用）。
func Kinds() []string {
	return []string{KindGithub, KindGitea, KindGitee, KindGeneric}
}

// ValidKind 判断 kind 是否合法。
func ValidKind(kind string) bool {
	for _, k := range Kinds() {
		if k == kind {
			return true
		}
	}
	return false
}

// NormalizeHost 归一化 host：去首尾空白、转小写、去尾部点号（DNS 全限定名形式）。
func NormalizeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

// ValidateHost 校验 host 是纯域名[:端口]——不含 scheme、路径、用户部分。
// 手误粘贴完整 URL（https://github.com/heyuuu）是最常见的配置错误，前置拦下。
func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("forge host 不得为空")
	}
	if strings.ContainsAny(host, "/\\ @") {
		return fmt.Errorf("forge host 须为纯域名（可带端口），不含协议/路径/用户部分: %s", host)
	}
	return nil
}

// RepoHost 解析 repo remote URL 的 host（兼容 git@host:path 与 https?://、ssh:// 形态），
// 归一化小写；无法解析返回空串（调用方按未匹配处理，不报错）。
func RepoHost(repoURL string) string {
	u, err := git.ParseRepoURL(repoURL)
	if err != nil || u == nil || u.Host == "" {
		return ""
	}
	// ssh://git@host:22/path 形态解析出的 Host 含端口，保留（与 forge host 声明口径一致）
	return NormalizeHost(u.Host)
}

// normalizeForge 归一化并校验一条待写入的 forge（save / save-all 共用），坏数据返回中文错误。
func normalizeForge(f Forge) (Forge, error) {
	f.Host = NormalizeHost(f.Host)
	if err := ValidateHost(f.Host); err != nil {
		return f, err
	}
	if !ValidKind(f.Kind) {
		return f, fmt.Errorf("forge kind 未知: %q（合法值：%s）", f.Kind, strings.Join(Kinds(), " / "))
	}
	if err := icons.ValidateIcon(f.Icon); err != nil {
		return f, fmt.Errorf("forge icon 配置错误: %s", err)
	}
	return f, nil
}

// forgeKey forge 的节内唯一键（归一化 host）。
func forgeKey(f Forge) string { return NormalizeHost(f.Host) }

// MatchHost 在 forges 里找 host 精确匹配的条目，无匹配返回 nil。
func MatchHost(forges []Forge, host string) *Forge {
	h := NormalizeHost(host)
	for i := range forges {
		if NormalizeHost(forges[i].Host) == h {
			return &forges[i]
		}
	}
	return nil
}
