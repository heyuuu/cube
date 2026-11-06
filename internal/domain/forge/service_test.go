package forge

import (
	"context"
	"testing"

	"cube/internal/testfixture"
	"cube/pkg/gitapi"
)

func newService(t *testing.T) (*Service, *testfixture.Workspace) {
	t.Helper()
	ws := testfixture.NewWorkspace(t)
	return NewService(ws.Cfg), ws
}

// TestSaveForge 新增与按 host 替换（归一化后同键）。
func TestSaveForge(t *testing.T) {
	s, _ := newService(t)
	if err := s.SaveForge(Forge{Host: "GitHub.COM", Kind: KindGithub}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if err := s.SaveForge(Forge{Host: "github.com", Kind: "unknown"}); err == nil {
		t.Fatal("未知 kind 应报错")
	}
	if len(s.Forges()) != 1 {
		t.Fatalf("同键替换后应只剩一条: %+v", s.Forges())
	}
}

// --- account（1041/1044） ---

// seedForge 写入一个 github.com forge，供 account 校验命中。
func seedForge(t *testing.T, s *Service) {
	t.Helper()
	if err := s.SaveForge(Forge{Host: "github.com", Kind: KindGithub}); err != nil {
		t.Fatalf("seed forge 失败: %v", err)
	}
}

// TestAccountCrud account 保存/替换/删除 + 校验（forge 未配置 / generic 拒配）+ 掩码语义。
func TestAccountCrud(t *testing.T) {
	s, _ := newService(t)

	if err := s.SaveAccount(Account{ForgeHost: "github.com", Username: "heyuuu", Token: "t1"}); err == nil {
		t.Fatal("forge 未配置时应报错")
	}
	seedForge(t, s)

	// generic forge 拒配
	if err := s.SaveForge(Forge{Host: "plain.example.com", Kind: KindGeneric}); err != nil {
		t.Fatalf("seed generic forge 失败: %v", err)
	}
	if err := s.SaveAccount(Account{ForgeHost: "plain.example.com", Username: "a"}); err == nil {
		t.Fatal("generic forge 应拒配 account")
	}

	if err := s.SaveAccount(Account{ForgeHost: "GitHub.COM", Username: "heyuuu", Token: "t1"}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if err := s.SaveAccount(Account{ForgeHost: "github.com", Username: "heyuuu", Token: "t2"}); err != nil {
		t.Fatalf("替换失败: %v", err)
	}
	accounts := s.Accounts()
	if len(accounts) != 1 || accounts[0].Token != "t2" || accounts[0].ForgeHost != "github.com" {
		t.Fatalf("account 状态不符: %+v", accounts)
	}

	// 掩码值 = 沿用旧 token；新账号提交掩码值报错
	if err := s.SaveAccount(Account{ForgeHost: "github.com", Username: "heyuuu", Token: TokenMasked}); err != nil {
		t.Fatalf("掩码提交应沿用旧值: %v", err)
	}
	if s.Accounts()[0].Token != "t2" {
		t.Fatalf("掩码提交后 token 不应变: %q", s.Accounts()[0].Token)
	}
	if err := s.SaveAccount(Account{ForgeHost: "github.com", Username: "new", Token: TokenMasked}); err == nil {
		t.Fatal("新账号提交掩码值应报错")
	}

	if err := s.DeleteAccount(Account{ForgeHost: "github.com", Username: "ghost"}); err == nil {
		t.Fatal("删除不存在的 account 应报错")
	}
	// 整项删除只按唯一键（归一化后）匹配，token 等其他字段不参与
	if err := s.DeleteAccount(Account{ForgeHost: " GitHub.com ", Username: "heyuuu", Token: TokenMasked}); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if len(s.Accounts()) != 0 {
		t.Fatal("删除后应为空")
	}
}

// TestDeleteForgeCascade 删 forge 级联清理该 host 下的 account。
func TestDeleteForgeCascade(t *testing.T) {
	s, _ := newService(t)
	seedForge(t, s)
	if err := s.SaveForge(Forge{Host: "gitea.example.com", Kind: KindGitea}); err != nil {
		t.Fatalf("seed 失败: %v", err)
	}
	for _, a := range []Account{
		{ForgeHost: "github.com", Username: "heyuuu", Token: "t"},
		{ForgeHost: "gitea.example.com", Username: "ops", Token: "t"},
	} {
		if err := s.SaveAccount(a); err != nil {
			t.Fatalf("保存 account 失败: %v", err)
		}
	}

	if err := s.DeleteForge(Forge{Host: "GitHub.com", Kind: KindGitea}); err != nil {
		t.Fatalf("删 forge 失败: %v", err)
	}
	if len(s.Accounts()) != 1 || s.Accounts()[0].ForgeHost != "gitea.example.com" {
		t.Fatalf("account 应只剩 gitea 一条: %+v", s.Accounts())
	}
}

// fakeClient 固定返回预置结果（List 调用计数供断言）。
type fakeClient struct {
	repos []gitapi.RemoteRepo
	lists int
}

func (f *fakeClient) ListAccountRepos(_ context.Context) ([]gitapi.RemoteRepo, error) {
	f.lists++
	return f.repos, nil
}

// TestFetchAccountPersist 拉取缓存（命中不重拉）+ 落盘（新 Service 同配置恢复不重拉）。
func TestFetchAccountPersist(t *testing.T) {
	s, ws := newService(t)
	seedForge(t, s)
	if err := s.SaveAccount(Account{ForgeHost: "github.com", Username: "heyuuu", Token: "t"}); err != nil {
		t.Fatalf("保存 account 失败: %v", err)
	}
	fake := &fakeClient{repos: []gitapi.RemoteRepo{{Name: "cube", CloneURL: "https://github.com/heyuuu/cube.git"}}}
	s.newClient = func(string, string, string) (gitapi.Client, error) { return fake, nil }

	repos, err := s.FetchAccount("github.com", "heyuuu", false)
	if err != nil || len(repos) != 1 {
		t.Fatalf("首次拉取失败: repos=%+v err=%v", repos, err)
	}
	if _, err := s.FetchAccount("github.com", "heyuuu", false); err != nil {
		t.Fatalf("二次拉取失败: %v", err)
	}
	if fake.lists != 1 {
		t.Fatalf("缓存命中不应重拉, lists=%d", fake.lists)
	}

	// 模拟重启：新 Service 指向同一落盘文件
	s2 := NewService(ws.Cfg)
	s2.newClient = s.newClient
	entry, ok := s2.CachedAccountRepos("github.com", "heyuuu")
	if !ok || len(entry.Repos) != 1 || entry.FetchedAt.IsZero() {
		t.Fatalf("重启后应从落盘恢复且带 fetchedAt: %+v ok=%v", entry, ok)
	}
	if _, err := s2.FetchAccount("github.com", "heyuuu", false); err != nil || fake.lists != 1 {
		t.Fatalf("落盘命中不应重拉: lists=%d err=%v", fake.lists, err)
	}
	if _, err := s.FetchAccount("github.com", "heyuuu", true); err != nil || fake.lists != 2 {
		t.Fatalf("force 应重拉: lists=%d err=%v", fake.lists, err)
	}
}

// TestSaveAllForges 整节覆盖：顺序即落盘序、未提交即删除（级联删 account）、
// 归一化后重复 / 单项非法整体拒绝不落盘、空清单清空。
func TestSaveAllForges(t *testing.T) {
	s, _ := newService(t)
	seedForge(t, s)
	if err := s.SaveForge(Forge{Host: "gitee.com", Kind: KindGitee}); err != nil {
		t.Fatalf("seed 失败: %v", err)
	}
	for _, a := range []Account{
		{ForgeHost: "github.com", Username: "a1", Token: "t"},
		{ForgeHost: "gitee.com", Username: "b1", Token: "t"},
	} {
		if err := s.SaveAccount(a); err != nil {
			t.Fatalf("保存 account 失败: %v", err)
		}
	}

	if err := s.SaveAllForges([]Forge{{Host: "gitee.com", Kind: KindGitee}, {Host: "new.example.com", Kind: KindGeneric}}); err != nil {
		t.Fatalf("save-all 失败: %v", err)
	}
	got := s.Forges()
	if len(got) != 2 || got[0].Host != "gitee.com" || got[1].Host != "new.example.com" {
		t.Fatalf("save-all 后 forge 不符: %+v", got)
	}
	if accounts := s.Accounts(); len(accounts) != 1 || accounts[0].ForgeHost != "gitee.com" {
		t.Fatalf("被移除 forge 的 account 应级联删除: %+v", accounts)
	}

	if err := s.SaveAllForges([]Forge{{Host: "a.com", Kind: KindGithub}, {Host: "A.com", Kind: KindGitea}}); err == nil {
		t.Fatal("归一化后重复 host 应报错")
	}
	if err := s.SaveAllForges([]Forge{{Host: "a.com", Kind: KindGithub}, {Host: "b.com", Kind: "bad"}}); err == nil {
		t.Fatal("单项非法应整体报错")
	}
	if got := s.Forges(); len(got) != 2 || got[0].Host != "gitee.com" {
		t.Fatalf("失败的 save-all 不应改动数据: %+v", got)
	}

	if err := s.SaveAllForges(nil); err != nil {
		t.Fatalf("空清单 save-all 失败: %v", err)
	}
	if len(s.Forges()) != 0 || len(s.Accounts()) != 0 {
		t.Fatalf("空清单应清空 forge 并级联清空 account: %+v %+v", s.Forges(), s.Accounts())
	}
}

// TestSaveAllAccounts 整节覆盖：顺序即落盘序、掩码 token 按唯一键沿用旧值、
// 新条目掩码 / forge 未配置 / 重复键整体拒绝。
func TestSaveAllAccounts(t *testing.T) {
	s, _ := newService(t)
	seedForge(t, s)
	if err := s.SaveForge(Forge{Host: "gitee.com", Kind: KindGitee}); err != nil {
		t.Fatalf("seed 失败: %v", err)
	}
	for _, a := range []Account{
		{ForgeHost: "github.com", Username: "a1", Token: "t1"},
		{ForgeHost: "github.com", Username: "a2", Token: "t2"},
		{ForgeHost: "gitee.com", Username: "b1", Token: "t3"},
	} {
		if err := s.SaveAccount(a); err != nil {
			t.Fatalf("保存 account 失败: %v", err)
		}
	}

	if err := s.SaveAllAccounts([]Account{
		{ForgeHost: "gitee.com", Username: "b1", Token: TokenMasked},
		{ForgeHost: "GitHub.com", Username: "a1", Token: TokenMasked},
	}); err != nil {
		t.Fatalf("save-all 失败: %v", err)
	}
	got := s.Accounts()
	if len(got) != 2 || got[0].Username != "b1" || got[0].Token != "t3" || got[1].Username != "a1" || got[1].Token != "t1" || got[1].ForgeHost != "github.com" {
		t.Fatalf("save-all 后 account 不符（掩码应沿用旧 token、未提交的 a2 应删除）: %+v", got)
	}

	for name, items := range map[string][]Account{
		"新条目掩码":     {{ForgeHost: "github.com", Username: "new", Token: TokenMasked}},
		"forge 未配置": {{ForgeHost: "nope.com", Username: "x", Token: "t"}},
		"重复键":       {{ForgeHost: "github.com", Username: "a1", Token: "t"}, {ForgeHost: "GITHUB.com", Username: "a1", Token: "t"}},
	} {
		if err := s.SaveAllAccounts(items); err == nil {
			t.Fatalf("%s 应报错", name)
		}
	}
	if len(s.Accounts()) != 2 {
		t.Fatalf("失败的 save-all 不应改动数据: %+v", s.Accounts())
	}
}
