package handlers

import (
	"strings"
	"testing"
)

// TestTplSourceListEmpty 空配置时 list 返回空数组（nil 切片序列化为 []）+ 默认源名。
func TestTplSourceListEmpty(t *testing.T) {
	env := newTestEnv(t)
	got := getJSON(t, env.url("/api/template/source/list"))
	var out TplSourceListResult
	decodeData(t, got, &out)
	if out.List == nil || len(out.List) != 0 {
		t.Fatalf("空配置应返回 []（nil 序列化）, got %#v", out.List)
	}
	if out.DefaultSource != "core" {
		t.Fatalf("defaultSource 应为常量 core 的值, got %q", out.DefaultSource)
	}
}

// TestTplSourceWrite save/delete/reorder 出口契约（含坏数据中文错误不落文件）。
func TestTplSourceWrite(t *testing.T) {
	env := newTestEnv(t)
	list := func() TplSourceListResult {
		t.Helper()
		var out TplSourceListResult
		decodeData(t, getJSON(t, env.url("/api/template/source/list")), &out)
		return out
	}

	// save 新增
	r := postJSON(t, env.url("/api/template/source/save"), `{"name":"core","repoUrl":"git@github.com:heyuuu/cube-templates.git"}`)
	if !r.Ok {
		t.Fatalf("save 应成功, message=%q", r.Message)
	}
	// save 第二条（不同名追加）
	if r := postJSON(t, env.url("/api/template/source/save"), `{"name":"work","repoUrl":"https://github.com/a/tpl.git"}`); !r.Ok {
		t.Fatalf("save work 应成功, message=%q", r.Message)
	}
	out := list()
	if len(out.List) != 2 || out.List[0].Name != "core" || out.List[0].RepoUrl != "git@github.com:heyuuu/cube-templates.git" {
		t.Fatalf("保存后列表不符: %+v", out.List)
	}

	// 同名替换（改 url 不追加）
	if r := postJSON(t, env.url("/api/template/source/save"), `{"name":"core","repoUrl":"git@github.com:heyuuu/cube-templates-v2.git"}`); !r.Ok {
		t.Fatalf("替换应成功, message=%q", r.Message)
	}
	out = list()
	if len(out.List) != 2 || out.List[0].RepoUrl != "git@github.com:heyuuu/cube-templates-v2.git" {
		t.Fatalf("同名应替换而非追加: %+v", out.List)
	}

	// save 坏数据：坏 name / 坏 repoUrl，中文错误、不落文件
	badName := postJSON(t, env.url("/api/template/source/save"), `{"name":"a/b","repoUrl":"git@github.com:a/b.git"}`)
	if badName.Ok || !strings.Contains(badName.Message, "source 名非法") {
		t.Fatalf("坏 name 应报中文错误, message=%q", badName.Message)
	}
	badUrl := postJSON(t, env.url("/api/template/source/save"), `{"name":"local","repoUrl":"github.com/a/b"}`)
	if badUrl.Ok || !strings.Contains(badUrl.Message, "repoUrl 不是合法地址") {
		t.Fatalf("坏 repoUrl 应报中文错误, message=%q", badUrl.Message)
	}
	if out := list(); len(out.List) != 2 {
		t.Fatalf("坏数据不应落文件, got %+v", out.List)
	}

	// reorder：work 提前，未列出条目殿后
	if r := postJSON(t, env.url("/api/template/source/reorder"), `{"names":["work"]}`); !r.Ok {
		t.Fatalf("reorder 应成功, message=%q", r.Message)
	}
	out = list()
	if len(out.List) != 2 || out.List[0].Name != "work" || out.List[1].Name != "core" {
		t.Fatalf("reorder 顺序不符: %+v", out.List)
	}
	// reorder 未知名中文错误
	r2 := postJSON(t, env.url("/api/template/source/reorder"), `{"names":["nope"]}`)
	if r2.Ok || !strings.Contains(r2.Message, "未找到指定") {
		t.Fatalf("reorder 未知名应报中文错误, message=%q", r2.Message)
	}

	// delete 按 name；再删报中文错误
	if r := postJSON(t, env.url("/api/template/source/delete"), `{"name":"core"}`); !r.Ok {
		t.Fatalf("delete 应成功, message=%q", r.Message)
	}
	r3 := postJSON(t, env.url("/api/template/source/delete"), `{"name":"core"}`)
	if r3.Ok || !strings.Contains(r3.Message, "未找到指定模板源") {
		t.Fatalf("删不存在应报中文错误, message=%q", r3.Message)
	}
}
