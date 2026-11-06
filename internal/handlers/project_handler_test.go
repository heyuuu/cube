package handlers

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"cube/internal/domain/project"
	"cube/internal/domain/project/projcache"
)

func TestProjectList(t *testing.T) {
	env := newTestEnv(t)
	var got struct {
		List []struct {
			Name    string           `json:"name"`
			Group   string           `json:"group"`
			Path    string           `json:"path"`
			Tags    []string         `json:"tags"`
			GitInfo *projcache.Entry `json:"gitInfo"`
		} `json:"list"`
	}
	decodeData(t, getJSON(t, env.url("/api/project/list")), &got)

	if len(got.List) != 2 {
		t.Fatalf("应扫描到 2 个项目, got %d: %+v", len(got.List), got.List)
	}
	byName := map[string]int{}
	for i, p := range got.List {
		byName[p.Name] = i
	}
	p1, ok := byName["g1:proj1"]
	if !ok {
		t.Fatalf("应包含 g1:proj1, got %+v", got.List)
	}
	if got.List[p1].Group != "g1" || got.List[p1].Path != env.proj1Path() {
		t.Errorf("g1:proj1 字段不符: %+v", got.List[p1])
	}
	// nil tags 序列化为 []（nilCollectionsJSONFormat 契约），gitInfo 未采集为 null
	if got.List[p1].Tags == nil {
		t.Error("nil tags 应序列化为 [] 而非 null")
	}
	if got.List[p1].GitInfo != nil {
		t.Errorf("未采集的 gitInfo 应为 null, got %+v", got.List[p1].GitInfo)
	}
	p2 := got.List[byName["g2:proj2"]]
	if len(p2.Tags) != 1 || p2.Tags[0] != "godot" {
		t.Errorf("g2:proj2 应带 godot tag, got %v", p2.Tags)
	}
}

func TestProjectScanRules(t *testing.T) {
	env := newTestEnv(t)
	var got struct {
		List []struct {
			Group    string `json:"group"`
			Path     string `json:"path"`
			MaxDepth int    `json:"maxDepth"`
		} `json:"list"`
	}
	decodeData(t, getJSON(t, env.url("/api/project/scan-rules")), &got)
	if len(got.List) != 2 {
		t.Fatalf("应返回 2 条 scan 规则, got %d", len(got.List))
	}
	// 配置里的路径在领域层已展开为绝对路径
	if got.List[0].Path != env.ws.Join("g1") || got.List[0].Group != "g1" || got.List[0].MaxDepth != 1 {
		t.Errorf("scan 规则字段不符: %+v", got.List[0])
	}
}

func TestProjectCloneRules(t *testing.T) {
	env := newTestEnv(t)
	var got struct {
		List []struct {
			RepoHost   string `json:"repoHost"`
			RepoPrefix string `json:"repoPrefix"`
			LocalPath  string `json:"localPath"`
		} `json:"list"`
	}
	decodeData(t, getJSON(t, env.url("/api/project/clone-rules")), &got)
	if len(got.List) != 1 {
		t.Fatalf("应返回 1 条 clone 规则, got %d", len(got.List))
	}
	if got.List[0].RepoHost != "github.com" || got.List[0].LocalPath != env.ws.Join("repo") {
		t.Errorf("clone 规则字段不符: %+v", got.List[0])
	}
}

// TestScanRuleWrite scan-rule 的 save/delete/save-all 出口契约（含坏数据中文错误）。
func TestScanRuleWrite(t *testing.T) {
	env := newTestEnv(t)
	root := env.ws.Mkdir("g3")

	// save 新增（带 icon + tags）
	r := postJSON(t, env.url("/api/project/scan-rule/save"), fmt.Sprintf(`{"group":"g3","path":%q,"maxDepth":2,"icon":{"type":"lucide","value":"folder-git-2"},"tags":["公司","开源","公司"," "]}`, root))
	if !r.Ok {
		t.Fatalf("save 应成功, message=%q", r.Message)
	}
	// save 坏数据：中文错误、不落文件
	bad := postJSON(t, env.url("/api/project/scan-rule/save"), `{"group":"","path":"/x","maxDepth":2}`)
	if bad.Ok || !strings.Contains(bad.Message, "group") {
		t.Fatalf("坏数据应报中文错误, ok=%v message=%q", bad.Ok, bad.Message)
	}
	// save 坏 icon：中文错误
	badIcon := postJSON(t, env.url("/api/project/scan-rule/save"), fmt.Sprintf(`{"group":"g4","path":%q,"maxDepth":2,"icon":{"type":"svg","value":"x"}}`, root))
	if badIcon.Ok || !strings.Contains(badIcon.Message, "icon type") {
		t.Fatalf("坏 icon 应报中文错误, message=%q", badIcon.Message)
	}

	list := getJSON(t, env.url("/api/project/scan-rules"))
	var got struct {
		List []project.ScanRule `json:"list"`
	}
	decodeData(t, list, &got)
	if len(got.List) != 3 { // fixture 预置 g1/g2 + 新增 g3
		t.Fatalf("应返回 3 条规则, got %d: %+v", len(got.List), got.List)
	}
	for _, r := range got.List { // icon/tags 落盘并回读（g3 带，g1/g2 未带为 nil）
		if r.Group == "g3" && (r.Icon == nil || r.Icon.Value != "folder-git-2") {
			t.Fatalf("g3 icon 未回读: %+v", r.Icon)
		}
		if r.Group == "g3" && !slices.Equal(r.Tags, []string{"公司", "开源"}) {
			t.Fatalf("g3 tags 应清洗为 [公司 开源]: %v", r.Tags)
		}
	}

	// save-all：list 行原样回交，g3 提到最前
	items := []project.ScanRule{}
	for _, r := range got.List {
		if r.Group == "g3" {
			items = append([]project.ScanRule{r}, items...)
		} else {
			items = append(items, r)
		}
	}
	body, _ := json.Marshal(map[string]any{"items": items})
	if r := postJSON(t, env.url("/api/project/scan-rule/save-all"), string(body)); !r.Ok {
		t.Fatalf("save-all 应成功, message=%q", r.Message)
	}
	decodeData(t, getJSON(t, env.url("/api/project/scan-rules")), &got)
	if len(got.List) != 3 || got.List[0].Group != "g3" {
		t.Fatalf("save-all 后 g3 应在最前: %+v", got.List)
	}
	// save-all group 重复：中文错误
	dupBody, _ := json.Marshal(map[string]any{"items": []project.ScanRule{items[0], {Group: "g3", Path: items[1].Path, MaxDepth: 1}}})
	if r := postJSON(t, env.url("/api/project/scan-rule/save-all"), string(dupBody)); r.Ok || !strings.Contains(r.Message, "group") {
		t.Fatalf("group 重复应报中文错误, message=%q", r.Message)
	}

	// delete：提交完整单项（按 path 定位）
	delBody, _ := json.Marshal(got.List[0])
	if r := postJSON(t, env.url("/api/project/scan-rule/delete"), string(delBody)); !r.Ok {
		t.Fatalf("delete 应成功, message=%q", r.Message)
	}
	decodeData(t, getJSON(t, env.url("/api/project/scan-rules")), &got)
	if len(got.List) != 2 {
		t.Fatalf("删除后应剩 2 条, got %d", len(got.List))
	}
	// delete 不存在：中文错误
	r2 := postJSON(t, env.url("/api/project/scan-rule/delete"), `{"group":"x","path":"/no/such","maxDepth":1}`)
	if r2.Ok || !strings.Contains(r2.Message, "未找到") {
		t.Fatalf("删除不存在应报中文错误, message=%q", r2.Message)
	}
}

// TestCloneRuleWrite clone-rule 的 save/delete/save-all 出口契约。
func TestCloneRuleWrite(t *testing.T) {
	env := newTestEnv(t)

	// save 新增（prefix 须以 / 开头）
	if r := postJSON(t, env.url("/api/project/clone-rule/save"), fmt.Sprintf(`{"repoHost":"gitee.com","repoPrefix":"/heyuuu","localPath":%q}`, env.ws.Dir)); !r.Ok {
		t.Fatalf("save 应成功, message=%q", r.Message)
	}
	// save 坏数据
	bad := postJSON(t, env.url("/api/project/clone-rule/save"), `{"repoHost":"gitee.com","repoPrefix":"bad","localPath":"/x"}`)
	if bad.Ok || !strings.Contains(bad.Message, "repoPrefix") {
		t.Fatalf("坏数据应报中文错误, ok=%v message=%q", bad.Ok, bad.Message)
	}

	var got struct {
		List []project.CloneRule `json:"list"`
	}
	decodeData(t, getJSON(t, env.url("/api/project/clone-rules")), &got)
	if len(got.List) != 2 { // fixture 预置 github.com + 新增 gitee
		t.Fatalf("应返回 2 条规则, got %d: %+v", len(got.List), got.List)
	}

	// save-all：list 行原样回交，gitee 提前
	body, _ := json.Marshal(map[string]any{"items": []project.CloneRule{got.List[1], got.List[0]}})
	if r := postJSON(t, env.url("/api/project/clone-rule/save-all"), string(body)); !r.Ok {
		t.Fatalf("save-all 应成功, message=%q", r.Message)
	}
	decodeData(t, getJSON(t, env.url("/api/project/clone-rules")), &got)
	if len(got.List) != 2 || got.List[0].RepoHost != "gitee.com" {
		t.Fatalf("save-all 后 gitee 应在最前: %+v", got.List)
	}

	// delete：提交完整单项（按 host+prefix 定位）
	if r := postJSON(t, env.url("/api/project/clone-rule/delete"), fmt.Sprintf(`{"repoHost":"gitee.com","repoPrefix":"/heyuuu","localPath":%q}`, env.ws.Dir)); !r.Ok {
		t.Fatalf("delete 应成功, message=%q", r.Message)
	}
	decodeData(t, getJSON(t, env.url("/api/project/clone-rules")), &got)
	if len(got.List) != 1 {
		t.Fatalf("删除后应剩 1 条, got %d", len(got.List))
	}
	// delete 不存在
	r2 := postJSON(t, env.url("/api/project/clone-rule/delete"), `{"repoHost":"nope.com","repoPrefix":"","localPath":"/x"}`)
	if r2.Ok || !strings.Contains(r2.Message, "未找到") {
		t.Fatalf("删除不存在应报中文错误, message=%q", r2.Message)
	}
}

func TestProjectOpen(t *testing.T) {
	env := newTestEnv(t)

	post := func(body string) envelope {
		resp, err := http.Post(env.url("/api/project/open"), "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST project/open 失败: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST project/open 应为 200, got %d", resp.StatusCode)
		}
		var env2 envelope
		if err := json.UnmarshalRead(resp.Body, &env2); err != nil {
			t.Fatalf("project/open 响应解析失败: %v", err)
		}
		return env2
	}

	// 打开项目成功：fakeExecutor 收到组装后的命令
	env1 := post(fmt.Sprintf(`{"path":%q,"opener":"finder"}`, env.proj1Path()))
	if !env1.Ok {
		t.Fatalf("打开项目应成功, message=%q", env1.Message)
	}
	if env.exec.callCount() != 1 || env.exec.lastCall()[0] != "/usr/bin/open" {
		t.Errorf("executor 应收到 finder 命令, calls=%v", env.exec.calls)
	}

	// 非收录项目路径
	env2 := post(`{"path":"/nonexistent/proj","opener":"finder"}`)
	if env2.Ok || !strings.Contains(env2.Message, "未找到指定项目") {
		t.Errorf("非收录路径应报错, got ok=%v message=%q", env2.Ok, env2.Message)
	}

	// 不存在的 opener
	env3 := post(fmt.Sprintf(`{"path":%q,"opener":"nope"}`, env.proj1Path()))
	if env3.Ok || !strings.Contains(env3.Message, "未找到指定 opener") {
		t.Errorf("未知 opener 应报错, got ok=%v message=%q", env3.Ok, env3.Message)
	}
}

func TestProjectList_LastUsedAt(t *testing.T) {
	env := newTestEnv(t)

	// 打开 proj2（原序列第二位），应产生 usage 记录
	body := fmt.Sprintf(`{"path":%q,"opener":"finder"}`, env.ws.Join("g2", "proj2"))
	resp, err := http.Post(env.url("/api/project/open"), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST project/open 失败: %v", err)
	}
	resp.Body.Close()

	var got struct {
		List []struct {
			Name       string     `json:"name"`
			LastUsedAt *time.Time `json:"lastUsedAt"`
		} `json:"list"`
	}
	decodeData(t, getJSON(t, env.url("/api/project/list")), &got)

	if len(got.List) != 2 {
		t.Fatalf("应扫描到 2 个项目, got %d", len(got.List))
	}
	// list 保持原始扫描序（排序是前端视图偏好），只补 lastUsedAt：proj2 有、proj1 无
	if got.List[0].Name != "g1:proj1" || got.List[1].Name != "g2:proj2" {
		t.Fatalf("list 应保持扫描原序, got %v", got.List)
	}
	if got.List[1].LastUsedAt == nil {
		t.Error("打开过的 g2:proj2 应带 lastUsedAt")
	}
	if got.List[0].LastUsedAt != nil {
		t.Errorf("未打开过的 g1:proj1 不应有 lastUsedAt, got %v", got.List[0])
	}
}

// TestWorkspaceGetSave workspace/get 三块信息（生效/声明/候选）+ save 全链路（写盘 + 即时重采集）。
func TestWorkspaceGetSave(t *testing.T) {
	env := newTestEnv(t)
	repo := env.proj1Path()

	// 建子目录 + pnpm 声明：未写 cube.json 时探测生效
	env.ws.Mkdir("g1/proj1/apps/web")
	if err := os.WriteFile(env.ws.Join("g1", "proj1", "pnpm-workspace.yaml"), []byte("packages:\n  - 'apps/*'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := env.projSvc.Refresh(); err != nil {
		t.Fatal(err)
	}

	// v2 解码字段名大小写敏感，这里直接用真实 DTO（匿名结构无 tag 时 Name 匹配不上 "name"）
	var state WorkspaceStateResult
	decodeData(t, getJSON(t, env.url("/api/project/workspace/get?path="+url.QueryEscape(repo))), &state)
	if state.DeclaredSet {
		t.Fatal("未写 cube.json 时 declaredSet 应为 false")
	}
	if len(state.Effective) != 1 || state.Effective[0].Path != "apps/web" {
		t.Fatalf("探测应生效于生效清单, got %+v", state.Effective)
	}
	if len(state.Detected) != 1 {
		t.Fatalf("探测候选应可得, got %+v", state.Detected)
	}

	// save 固化显式声明 → get 反映 declaredSet，且 effective 即时刷新（不等 TTL）
	body := fmt.Sprintf(`{"path":%q,"workspaces":[{"name":"前端","path":"apps/web"}]}`, repo)
	resp, err := http.Post(env.url("/api/project/workspace/save"), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var saved envelope
	if err := json.UnmarshalRead(resp.Body, &saved); err != nil || !saved.Ok {
		t.Fatalf("save 应成功: err=%v env=%+v", err, saved)
	}

	decodeData(t, getJSON(t, env.url("/api/project/workspace/get?path="+url.QueryEscape(repo))), &state)
	if !state.DeclaredSet {
		t.Fatal("save 后 declaredSet 应为 true")
	}
	if len(state.Effective) != 1 || state.Effective[0].Name != "前端" {
		t.Fatalf("save 后 effective 应即时反映显式声明, got %+v", state.Effective)
	}

	// 坏条目整体拒绝（写侧严格语义，区别于采集侧静默跳过）
	bad := fmt.Sprintf(`{"path":%q,"workspaces":[{"name":"逃逸","path":"../outside"}]}`, repo)
	resp2, err := http.Post(env.url("/api/project/workspace/save"), "application/json", strings.NewReader(bad))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var rejected envelope
	if err := json.UnmarshalRead(resp2.Body, &rejected); err != nil {
		t.Fatal(err)
	}
	if rejected.Ok {
		t.Fatal("坏条目应整体拒绝")
	}
}
