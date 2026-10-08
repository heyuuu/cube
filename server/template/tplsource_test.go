package template

import (
	"strings"
	"testing"

	"cube/core/config"
	"cube/internal/testfixture"
)

// newConfigService 建配置管理测试 Service（settings.json 落临时目录）。
func newConfigService(t *testing.T) *Service {
	t.Helper()
	ws := testfixture.NewWorkspace(t)
	return NewService(ws.Join("settings.json"), ws.Join("tpl"))
}

// TestValidateTplSource name / repoUrl 校验表。
func TestValidateTplSource(t *testing.T) {
	cases := []struct {
		name    string
		src     TplSource
		wantErr string // 空 = 应通过
	}{
		{"合法 ssh 地址", TplSource{Name: "core", RepoUrl: "git@github.com:heyuuu/cube-templates.git"}, ""},
		{"合法 https 地址", TplSource{Name: "work-2_x", RepoUrl: "https://github.com/a/b.git"}, ""},
		{"name 含路径分隔符", TplSource{Name: "a/b", RepoUrl: "git@github.com:a/b.git"}, "source 名非法"},
		{"name 相对逃逸", TplSource{Name: "..", RepoUrl: "git@github.com:a/b.git"}, "source 名非法"},
		{"name 空白", TplSource{Name: " ", RepoUrl: "git@github.com:a/b.git"}, "source 名非法"},
		{"repoUrl 本地路径", TplSource{Name: "core", RepoUrl: "/local/path"}, "repoUrl 不是合法地址"},
		{"repoUrl 裸 host 路径", TplSource{Name: "core", RepoUrl: "github.com/a/b"}, "repoUrl 不是合法地址"},
		{"repoUrl 空", TplSource{Name: "core", RepoUrl: ""}, "repoUrl 不是合法地址"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateTplSource(c.src)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("应通过, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("期望错误含 %q, got %v", c.wantErr, err)
			}
		})
	}
}

// TestTplSourceCRUD save 同名替换 / delete / reorder 的配置节语义。
func TestTplSourceCRUD(t *testing.T) {
	s := newConfigService(t)

	// 空配置返回空列表
	if got := s.TplSources(); len(got) != 0 {
		t.Fatalf("空配置应无条目, got %+v", got)
	}

	// save 两条
	if err := s.SaveTplSource(TplSource{Name: "core", RepoUrl: "git@github.com:a/core.git"}); err != nil {
		t.Fatalf("save core 失败: %v", err)
	}
	if err := s.SaveTplSource(TplSource{Name: "work", RepoUrl: "https://github.com/a/work.git"}); err != nil {
		t.Fatalf("save work 失败: %v", err)
	}

	// 同名替换而非追加（name 是唯一键）
	if err := s.SaveTplSource(TplSource{Name: "core", RepoUrl: "git@github.com:a/core-v2.git"}); err != nil {
		t.Fatalf("替换 core 失败: %v", err)
	}
	got := s.TplSources()
	if len(got) != 2 || got[0].RepoUrl != "git@github.com:a/core-v2.git" {
		t.Fatalf("同名应原位替换: %+v", got)
	}

	// 坏数据不落文件
	if err := s.SaveTplSource(TplSource{Name: "a/b", RepoUrl: "git@github.com:a/b.git"}); err == nil || !strings.Contains(err.Error(), "source 名非法") {
		t.Fatalf("坏 name 应报中文错误, got %v", err)
	}
	if err := s.SaveTplSource(TplSource{Name: "bad-url", RepoUrl: "/local/path"}); err == nil || !strings.Contains(err.Error(), "repoUrl 不是合法地址") {
		t.Fatalf("坏 repoUrl 应报中文错误, got %v", err)
	}
	if len(s.TplSources()) != 2 {
		t.Fatalf("坏数据不应落文件, got %+v", s.TplSources())
	}

	// reorder：work 提前，未列出的殿后
	if err := s.ReorderTplSources([]string{"work"}); err != nil {
		t.Fatalf("reorder 失败: %v", err)
	}
	got = s.TplSources()
	if len(got) != 2 || got[0].Name != "work" || got[1].Name != "core" {
		t.Fatalf("reorder 顺序不符: %+v", got)
	}

	// reorder 未知名报中文错误
	if err := s.ReorderTplSources([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "未找到指定") {
		t.Fatalf("reorder 未知名应报错, got %v", err)
	}

	// delete + 再删报错
	if err := s.DeleteTplSource("core"); err != nil {
		t.Fatalf("delete 失败: %v", err)
	}
	if err := s.DeleteTplSource("core"); err == nil || !strings.Contains(err.Error(), "未找到指定模板源") {
		t.Fatalf("删不存在应报中文错误, got %v", err)
	}
	if len(s.TplSources()) != 1 {
		t.Fatalf("删除后应剩一条, got %+v", s.TplSources())
	}
}

// TestTplSourcesSkipBad 读侧跳过坏条目不阻断（手编 settings.json 容错）。
func TestTplSourcesSkipBad(t *testing.T) {
	s := newConfigService(t)
	var raw []map[string]string
	raw = append(raw,
		map[string]string{"name": "core", "repoUrl": "git@github.com:a/core.git"},
		map[string]string{"name": "a/b", "repoUrl": "git@github.com:a/b.git"}, // 坏 name
		map[string]string{"name": "no-url", "repoUrl": ""},                    // 坏 url
	)
	if err := config.SaveSection(s.settingsFile, tplSourcesSection, raw); err != nil {
		t.Fatalf("写入原始配置失败: %v", err)
	}

	got := s.TplSources()
	if len(got) != 1 || got[0].Name != "core" {
		t.Fatalf("坏条目应跳过只留 core, got %+v", got)
	}
}
