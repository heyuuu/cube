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

// Account 一条 forge 账号配置，纯 API 凭证（token 只用于调平台 API 拉仓库列表，
// 不参与 git 传输——git 操作走本机凭证）。forgeHost+username 归一化后是唯一键。
type Account struct {
	ForgeHost string `json:"forgeHost"` // 所属 forge host，必须命中已配置 forge
	Username  string `json:"username"`  // 平台用户名
	Token     string `json:"token"`     // API token，可为空（仅公开数据）
}

// TokenMasked token 的掩码展示串：list 端点用返回，save 时提交此值视为「未修改」沿用旧值。
const TokenMasked = "••••••"

// NormalizeUsername 归一化用户名：去首尾空白。保留大小写（部分平台用户名大小写敏感）。
func NormalizeUsername(username string) string { return strings.TrimSpace(username) }
