package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitTemplateMeta(t *testing.T) {
	valid := `
version: 1
variables:
  - name: project-name
    prompt: 项目名
    required: true
  - name: author
    prompt: 作者
    default: heyu
patterns:
  - pattern: __MODULE__
    replace: "github.com/${author}/${project-name}"
    scope: "**/*.go"
init:
  - git init -b master
`
	tpl, err := InitTemplateMeta([]byte(valid))
	if err != nil {
		t.Fatalf("解析合法 yaml 报错: %v", err)
	}
	if tpl.Version != 1 {
		t.Fatalf("version = %d, want 1", tpl.Version)
	}

	// 校验 variables
	wantVariables := []VariableDecl{
		{
			Name:     "project-name",
			Prompt:   "项目名",
			Required: true,
			Default:  "",
		},
		{
			Name:     "author",
			Prompt:   "作者",
			Required: false,
			Default:  "heyu",
		},
	}
	assert.Equalf(t, wantVariables, tpl.Variables, "variables 解析结果不符: %+v", tpl.Variables)

	// 校验 Patterns
	wantPatterns := []PatternDecl{
		{
			Pattern: "__MODULE__",
			Replace: "github.com/${author}/${project-name}",
			Scope:   "**/*.go",
		},
	}
	assert.Equalf(t, wantPatterns, tpl.Patterns, "Patterns 解析结果不符: %+v", tpl.Patterns)

	// 校验 Init
	wantInit := []string{
		"git init -b master",
	}
	assert.Equalf(t, wantInit, tpl.Init, "Init 解析结果不符: %+v", tpl.Init)
}

func TestInitTemplateMeta_wantError(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{"缺少 version", "variables: []", "version"},
		{"版本过高", "version: 99", "高于引擎支持"},
		{"坏缩进", "version: 1\nvariables:\n  a:\n   b:\n   c: 1\n  d", "解析失败"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := InitTemplateMeta([]byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("期望错误含 %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestInitTemplateMeta_ignoreUnknown(t *testing.T) {
	t.Run("未知字段忽略", func(t *testing.T) {
		_, err := InitTemplateMeta([]byte("version: 1\nfuture-field: whatever\n"))
		if err != nil {
			t.Fatalf("未知字段应忽略: %v", err)
		}
	})
}

func TestInterpolate(t *testing.T) {
	vars := map[string]string{"name": "my-app", "author": "heyu"}
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"无引用", "plain text", "plain text", false},
		{"单变量", "${name}", "my-app", false},
		{"多变量混排", "github.com/${author}/${name}", "github.com/heyu/my-app", false},
		{"同一变量多次", "${name}/${name}.go", "my-app/my-app.go", false},
		{"未闭合不算引用", "${name", "${name", false},
		{"未定义变量", "${nope}", "", true},
		{"混合未定义", "${author}/${nope}", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interpolate(tt.input, vars)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望报错，得到 %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外报错: %v", err)
			}
			if got != tt.want {
				t.Fatalf("interpolate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
