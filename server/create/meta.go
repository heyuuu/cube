package create

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
	"go.yaml.in/yaml/v3"
)

const MetaFileName = "template.yaml"

// TemplateVersionCurrent 是当前引擎支持的 template.yaml 协议版本。
const TemplateVersionCurrent = 1

// TemplateMeta 是模板根目录 MetaFile 的解析结果，引擎与模板之间的唯一契约。
type TemplateMeta struct {
	Version   int            `yaml:"version"`
	Variables []VariableDecl `yaml:"variables"`
	Patterns  []PatternDecl  `yaml:"patterns"`
	Init      []string       `yaml:"init"`
}

// VariableDecl 声明一个需要收集的输入变量。
type VariableDecl struct {
	Name     string `yaml:"name"`
	Prompt   string `yaml:"prompt"`
	Required bool   `yaml:"required"`
	Default  string `yaml:"default"`
}

// PatternDecl 是一条精确字符串替换规则，同时作用于文件路径与内容。
type PatternDecl struct {
	Pattern string `yaml:"pattern"`
	Replace string `yaml:"replace"`
	Scope   string `yaml:"scope"`
}

func (p PatternDecl) MatchScope(relPath string) bool {
	if p.Scope == "" {
		return true
	}
	ok, _ := doublestar.Match(p.Scope, relPath)
	return ok
}

// InitTemplateMeta 解析 template.yaml 内容。未知字段忽略（协议留白字段多，宽容优先），
// yaml 的报错自带行号，直接透传给模板作者。
func InitTemplateMeta(data []byte) (*TemplateMeta, error) {
	var tpl TemplateMeta
	if err := yaml.Unmarshal(data, &tpl); err != nil {
		return nil, fmt.Errorf("解析失败: %w", err)
	}
	if tpl.Version == 0 {
		return nil, fmt.Errorf("缺少 version 字段（当前协议版本为 %d）", TemplateVersionCurrent)
	}
	if tpl.Version > TemplateVersionCurrent {
		return nil, fmt.Errorf("版本 %d 高于引擎支持的版本 %d，请升级 cube", tpl.Version, TemplateVersionCurrent)
	}
	return &tpl, nil
}

// varRefPattern 匹配 ${var} 形式的变量引用，只在 template.yaml 的值内生效。
var varRefPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// interpolate 对字符串做 ${var} → 实际值替换，引用了未收集的变量时报错
// （拼写错误应尽早暴露，而不是静默留在生成结果里）。
func interpolate(s string, vars map[string]string) (string, error) {
	var unknown []string
	out := varRefPattern.ReplaceAllStringFunc(s, func(ref string) string {
		name := ref[2 : len(ref)-1]
		if v, ok := vars[name]; ok {
			return v
		}
		unknown = append(unknown, name)
		return ref
	})
	if len(unknown) > 0 {
		return "", fmt.Errorf("引用了未定义的变量: %v", unknown)
	}
	return out, nil
}

// interpolatePatterns 对 TemplateMeta.Patterns 做插值运算
func interpolatePatterns(patterns []PatternDecl, vars map[string]string) ([]PatternDecl, error) {
	result := slices.Clone(patterns) // 复制，不影响原数据
	for i, rule := range result {
		v, err := interpolate(rule.Replace, vars)
		if err != nil {
			return nil, fmt.Errorf("patterns[%d]: %w", i, err)
		}
		result[i].Replace = v
	}
	return result, nil
}

// interpolateInit 对 TemplateMeta.Init 做插值运算
func interpolateInit(init []string, vars map[string]string) ([]string, error) {
	result := slices.Clone(init) // 复制，不影响原数据
	for i, cmd := range result {
		v, err := interpolate(cmd, vars)
		if err != nil {
			return nil, fmt.Errorf("init[%d]: %w", i, err)
		}
		result[i] = v
	}
	return result, nil
}
