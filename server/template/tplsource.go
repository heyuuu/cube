package template

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"cube/util/git"
)

// settings.json 中的模板源节名。
const tplSourcesSection = "tplSources"

// TplSource 模板源配置：source 名 ↔ git 仓库地址的映射（settings.json tplSources 节）。
// name 是 cube create 的引用短名（@name/模板名），节内顺序即 create 交互的选择顺序。
type TplSource struct {
	Name    string `json:"name"`
	RepoUrl string `json:"repoUrl"`
}

// tplSourceNamePattern source 名约束：字母或数字开头，仅含字母/数字/-/_，最长 32。
// name 会进入 CLI 引用语法（@name/tpl），禁路径分隔符与空白等歧义字符。
var tplSourceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,31}$`)

// ValidateTplSource 校验一条模板源（写侧收敛，坏数据中文错误不落文件）。
func ValidateTplSource(src TplSource) error {
	if !tplSourceNamePattern.MatchString(strings.TrimSpace(src.Name)) {
		return fmt.Errorf("source 名非法: %q（需字母或数字开头，仅含字母/数字/-/_，最长 32）", src.Name)
	}
	if err := validateTplSourceUrl(src.RepoUrl); err != nil {
		return err
	}
	return nil
}

// validateTplSourceUrl 校验 repoUrl 是可 clone 的 git 地址：git@host:path / https://host:path /
// 本地绝对路径（模板仓库放本地盘也是正当用法，git 原生支持 clone 本地路径）。
// ParseRepoUrl 对裸路径/裸 host 也解析成功（scheme/host 为空），这里按形态收紧。
func validateTplSourceUrl(rawUrl string) error {
	trimmed := strings.TrimSpace(rawUrl)
	if filepath.IsAbs(trimmed) {
		return nil
	}
	u, err := git.ParseRepoUrl(trimmed)
	if err != nil {
		return fmt.Errorf("repoUrl 不是合法地址: url=%s", rawUrl)
	}
	switch u.Scheme {
	case "git", "https", "http":
	default:
		return fmt.Errorf("repoUrl 不是合法地址: url=%s（支持 git@host:path、https://host/path 或本地绝对路径）", rawUrl)
	}
	if u.Host == "" || u.Path == "" {
		return fmt.Errorf("repoUrl 不是合法地址: url=%s（缺少 host 或仓库路径）", rawUrl)
	}
	return nil
}
