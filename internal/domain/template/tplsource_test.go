package template

import (
	"strings"
	"testing"

	"cube/internal/testfixture"
)

// newConfigService 建配置管理测试 Service（settings.json 落临时目录）。
func newConfigService(t *testing.T) (*Service, *testfixture.Workspace) {
	t.Helper()
	ws := testfixture.NewWorkspace(t)
	return NewService(ws.Cfg), ws
}

// TestValidateTplSource name / repoUrl 校验表。
func TestValidateTplSource(t *testing.T) {
	cases := []struct {
		name    string
		src     TplSource
		wantErr string // 空 = 应通过
	}{
		{"合法 ssh 地址", TplSource{Name: "core", RepoURL: "git@github.com:heyuuu/cube-templates.git"}, ""},
		{"合法 https 地址", TplSource{Name: "work-2_x", RepoURL: "https://github.com/a/b.git"}, ""},
		{"合法本地绝对路径", TplSource{Name: "local", RepoURL: "/local/path/tpl.git"}, ""},
		{"name 含路径分隔符", TplSource{Name: "a/b", RepoURL: "git@github.com:a/b.git"}, "source 名非法"},
		{"name 相对逃逸", TplSource{Name: "..", RepoURL: "git@github.com:a/b.git"}, "source 名非法"},
		{"name 空白", TplSource{Name: " ", RepoURL: "git@github.com:a/b.git"}, "source 名非法"},
		{"repoUrl 裸 host 路径", TplSource{Name: "core", RepoURL: "github.com/a/b"}, "repoUrl 不是合法地址"},
		{"repoUrl 空", TplSource{Name: "core", RepoURL: ""}, "repoUrl 不是合法地址"},
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

// TestTplSourceCRUD save 同名替换 / delete / save-all 的配置节语义。
func TestTplSourceCRUD(t *testing.T) {
	s, _ := newConfigService(t)

	// 空配置返回空列表
	if got := s.TplSources(); len(got) != 0 {
		t.Fatalf("空配置应无条目, got %+v", got)
	}

	// save 两条
	if err := s.SaveTplSource(TplSource{Name: "core", RepoURL: "git@github.com:a/core.git"}); err != nil {
		t.Fatalf("save core 失败: %v", err)
	}
	if err := s.SaveTplSource(TplSource{Name: "work", RepoURL: "https://github.com/a/work.git"}); err != nil {
		t.Fatalf("save work 失败: %v", err)
	}

	// 同名替换而非追加（name 是唯一键）
	if err := s.SaveTplSource(TplSource{Name: "core", RepoURL: "git@github.com:a/core-v2.git"}); err != nil {
		t.Fatalf("替换 core 失败: %v", err)
	}
	got := s.TplSources()
	if len(got) != 2 || got[0].RepoURL != "git@github.com:a/core-v2.git" {
		t.Fatalf("同名应原位替换: %+v", got)
	}

	// 坏数据不落文件
	if err := s.SaveTplSource(TplSource{Name: "a/b", RepoURL: "git@github.com:a/b.git"}); err == nil || !strings.Contains(err.Error(), "source 名非法") {
		t.Fatalf("坏 name 应报中文错误, got %v", err)
	}
	if err := s.SaveTplSource(TplSource{Name: "bad-url", RepoURL: "github.com/a/b"}); err == nil || !strings.Contains(err.Error(), "repoUrl 不是合法地址") {
		t.Fatalf("坏 repoUrl 应报中文错误, got %v", err)
	}
	if len(s.TplSources()) != 2 {
		t.Fatalf("坏数据不应落文件, got %+v", s.TplSources())
	}

	// save-all：提交清单即落盘结果（顺序 + 未提交即删除 + 新增），name 归一化
	if err := s.SaveAllTplSources([]TplSource{
		{Name: "work", RepoURL: "https://github.com/a/work.git"},
		{Name: " extra ", RepoURL: "git@github.com:a/extra.git"},
		{Name: "core", RepoURL: "git@github.com:a/core-v2.git"},
	}); err != nil {
		t.Fatalf("save-all 失败: %v", err)
	}
	got = s.TplSources()
	if len(got) != 3 || got[0].Name != "work" || got[1].Name != "extra" || got[2].Name != "core" {
		t.Fatalf("save-all 顺序不符: %+v", got)
	}

	// save-all 重复名 / 单项非法整体拒绝不落盘
	if err := s.SaveAllTplSources([]TplSource{{Name: "core", RepoURL: "git@github.com:a/x.git"}, {Name: "core ", RepoURL: "git@github.com:a/y.git"}}); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("save-all 重复名应报错, got %v", err)
	}
	if err := s.SaveAllTplSources([]TplSource{{Name: "core", RepoURL: "git@github.com:a/x.git"}, {Name: "a/b", RepoURL: "git@github.com:a/y.git"}}); err == nil {
		t.Fatal("save-all 单项非法应报错")
	}
	if len(s.TplSources()) != 3 {
		t.Fatalf("失败的 save-all 不应改动数据, got %+v", s.TplSources())
	}
	if err := s.SaveAllTplSources([]TplSource{got[0], got[2]}); err != nil {
		t.Fatalf("save-all 失败: %v", err)
	}

	// delete + 再删幂等成功
	if err := s.DeleteTplSource(TplSource{Name: "core", RepoURL: "旧地址不参与匹配"}); err != nil {
		t.Fatalf("delete 失败: %v", err)
	}
	if err := s.DeleteTplSource(TplSource{Name: "core"}); err != nil {
		t.Fatalf("删不存在应幂等成功, got %v", err)
	}
	if len(s.TplSources()) != 1 {
		t.Fatalf("删除后应剩一条, got %+v", s.TplSources())
	}
}

// TestTplSourcesSkipBad 读侧跳过坏条目不阻断（手编 settings.json 容错）。
func TestTplSourcesSkipBad(t *testing.T) {
	s, ws := newConfigService(t)
	ws.SaveSection(tplSourcesSection, []map[string]string{
		{"name": "core", "repoUrl": "git@github.com:a/core.git"},
		{"name": "a/b", "repoUrl": "git@github.com:a/b.git"}, // 坏 name
		{"name": "no-url", "repoUrl": ""},                    // 坏 url
	})

	got := s.TplSources()
	if len(got) != 1 || got[0].Name != "core" {
		t.Fatalf("坏条目应跳过只留 core, got %+v", got)
	}
}
