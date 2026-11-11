package opener

import (
	"os"
	"reflect"
	"testing"
)

func mkmk(pairs ...any) map[Role]string {
	m := map[Role]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i].(Role)] = "exec: " + pairs[i+1].(string)
	}
	return m
}

// ---------- renderArgs ----------

func TestRenderArgs(t *testing.T) {
	paths := []string{"/a", "/b"}
	cases := []struct {
		name string
		args []string
		want []string
		used int
	}{
		{"无占位符原样保留", []string{"-old", "/x"}, []string{"-old", "/x"}, 0},
		{"单占位符", []string{"$0"}, []string{"/a"}, 1},
		{"flag + 占位符混合", []string{"-old", "$0", "-new", "$1"}, []string{"-old", "/a", "-new", "/b"}, 2},
		{"占位符乱序", []string{"$1", "$0"}, []string{"/b", "/a"}, 2},
		{"占位符越界保留原样", []string{"$0", "$9"}, []string{"/a", "$9"}, 1},
		{"token 内嵌占位符", []string{"--wd=$0", "x$1y"}, []string{"--wd=/a", "x/by"}, 2},
		{"多位数字占位符整体识别", []string{"$10"}, []string{"$10"}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, used := renderArgs(c.args, paths)
			if !reflect.DeepEqual(got, c.want) || used != c.used {
				t.Fatalf("renderArgs(%v) = (%v,%d), want (%v,%d)", c.args, got, used, c.want, c.used)
			}
		})
	}
}

// ---------- buildActionArgs ----------

func TestBuildActionArgs(t *testing.T) {
	cases := []struct {
		name     string
		actions  map[Role]string
		role     Role
		paths    []string
		wantBin  string
		wantArgs []string
	}{
		{
			name:     "bin 在前 + 占位符参数",
			actions:  mkmk(RoleOpenDir, "code $0"),
			role:     RoleOpenDir,
			paths:    []string{"/p"},
			wantBin:  "code",
			wantArgs: []string{"/p"},
		},
		{
			name:     "含空格路径引号包裹",
			actions:  mkmk(RoleOpenDir, `"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" $0`),
			role:     RoleOpenDir,
			paths:    []string{"/p"},
			wantBin:  "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			wantArgs: []string{"/p"},
		},
		{
			name:     "token 内嵌占位符",
			actions:  mkmk(RoleOpenDir, "tool --wd=$0"),
			role:     RoleOpenDir,
			paths:    []string{"/p"},
			wantBin:  "tool",
			wantArgs: []string{"--wd=/p"},
		},
		{
			name:     "cmd[0] 本身是占位符（用路径作可执行文件）",
			actions:  mkmk(RoleOpenFile, "$0"),
			role:     RoleOpenFile,
			paths:    []string{"/bin/sh"},
			wantBin:  "/bin/sh",
			wantArgs: []string{},
		},
		{
			name:     "无占位符：路径追加到末尾",
			actions:  mkmk(RoleOpenDir, "code"),
			role:     RoleOpenDir,
			paths:    []string{"/p"},
			wantBin:  "code",
			wantArgs: []string{"/p"},
		},
		{
			name:     "对比工具：双槽",
			actions:  mkmk(RoleDiffDir, "bcompare $0 $1", RoleDiffFile, "bcompare $0 $1"),
			role:     RoleDiffDir,
			paths:    []string{"/a", "/b"},
			wantBin:  "bcompare",
			wantArgs: []string{"/a", "/b"},
		},
		{
			name: "同 opener 不同 role 不同 cmd（vscode 场景，本次改造核心动机）",
			actions: mkmk(
				RoleOpenDir, "code $0",
				RoleDiffFile, "code --diff $0 $1",
			),
			role:     RoleDiffFile,
			paths:    []string{"/a", "/b"},
			wantBin:  "code",
			wantArgs: []string{"--diff", "/a", "/b"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bin, args, err := buildActionArgs(c.role, c.actions[c.role], c.paths, "")
			if err != nil {
				t.Fatalf("buildActionArgs 失败: %v", err)
			}
			if bin != c.wantBin || !reflect.DeepEqual(args, c.wantArgs) {
				t.Fatalf("buildActionArgs(%v) = (%q,%v), want (%q,%v)", c.paths, bin, args, c.wantBin, c.wantArgs)
			}
		})
	}
}

// ---------- cmd[0] 自引用替换（resolveBin） ----------

func TestBuildActionArgsResolvesSelfCube(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable 失败: %v", err)
	}
	cases := []struct {
		name     string
		cmd      string
		wantArgs []string
	}{
		{"裸名 cube", "cube ui workbench $0", []string{"ui", "workbench", "/proj"}},
		{"绝对路径 cube", "/usr/local/bin/cube md", []string{"md", "/proj"}},
		{"相对路径 cube", "tmp/cube md", []string{"md", "/proj"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bin, args, err := buildActionArgs(RoleOpenDir, "exec: "+c.cmd, []string{"/proj"}, "")
			if err != nil {
				t.Fatalf("buildActionArgs 失败: %v", err)
			}
			if bin != self {
				t.Fatalf("cmd 应替换为当前可执行文件 %q, got %q", self, bin)
			}
			if !reflect.DeepEqual(args, c.wantArgs) {
				t.Fatalf("args = %v, want %v", args, c.wantArgs)
			}
		})
	}

	t.Run("近似名不替换", func(t *testing.T) {
		for _, bin0 := range []string{"cubed", "cube-x", "mycube", "/bin/cubecase"} {
			bin, _, err := buildActionArgs(RoleOpenDir, "exec: "+bin0, []string{"/proj"}, "")
			if err != nil {
				t.Fatalf("buildActionArgs 失败: %v", err)
			}
			if bin != bin0 {
				t.Fatalf("%q 不应被替换, got %q", bin0, bin)
			}
		}
	})
}

// ---------- tokenizeCmd ----------

func TestTokenizeCmd(t *testing.T) {
	cases := []struct {
		name string
		line string
		want []string
	}{
		{"空串", "", nil},
		{"纯空白", "   ", nil},
		{"空格分隔", "code $0 -n", []string{"code", "$0", "-n"}},
		{"多空白折叠", "code   $0", []string{"code", "$0"}},
		{"双引号含空格", `"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" $0`, []string{"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code", "$0"}},
		{"单引号含空格", `'a b' c`, []string{"a b", "c"}},
		{"双引号内转义引号", `"a\"b"`, []string{`a"b`}},
		{"裸词反斜杠转义", `a\ b`, []string{"a b"}},
		{"tab 分隔", "a\tb", []string{"a", "b"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := tokenizeCmd(c.line)
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Fatalf("tokenizeCmd(%q) = (%v,%v), want (%v,nil)", c.line, got, err, c.want)
			}
		})
	}
	// 引号未闭合应报错（保存/加载边界拦截，不拖到运行期 exec 失败）
	for _, line := range []string{`"abc`, `'abc`, `a "b c`, `"a\"`} {
		if _, err := tokenizeCmd(line); err == nil {
			t.Fatalf("tokenizeCmd(%q) 引号未闭合应报错", line)
		}
	}
}

// ---------- scanPlaceholders ----------

func TestScanPlaceholders(t *testing.T) {
	cases := []struct {
		token string
		want  []int
	}{
		{"$0", []int{0}},
		{"--wd=$1", []int{1}},
		{"a$0b$1", []int{0, 1}},
		{"$12", []int{12}},
		{"$x", nil},
		{"no placeholder", nil},
		{"$", nil},
	}
	for _, c := range cases {
		if got := scanPlaceholders(c.token); !reflect.DeepEqual(got, c.want) {
			t.Fatalf("scanPlaceholders(%q) = %v, want %v", c.token, got, c.want)
		}
	}
}

// ---------- url 动作（站内路由 / 外部 URL） ----------

func TestURLActions(t *testing.T) {
	const base = "http://127.0.0.1:6001"
	cases := []struct {
		name    string
		actions map[Role]string
		role    Role
		paths   []string
		wantURL string
	}{
		{
			name:    "站内路由：拼 baseURL + 占位符 query encode",
			actions: map[Role]string{RoleOpenDir: "url:/workbench?path=$0"},
			role:    RoleOpenDir,
			paths:   []string{"/Users/x/My Proj"},
			wantURL: base + "/workbench?path=" + "%2FUsers%2Fx%2FMy+Proj",
		},
		{
			name:    "外部 URL：占位符照常替换（不 encode）",
			actions: map[Role]string{RoleOpenDir: "url: https://github.com/search?q=$0"},
			role:    RoleOpenDir,
			paths:   []string{"/a/b"},
			wantURL: "https://github.com/search?q=/a/b",
		},
		{
			name:    "外部 URL 无占位符原样打开",
			actions: map[Role]string{RoleOpenDir: "url: https://github.com"},
			role:    RoleOpenDir,
			paths:   []string{"/ignored"},
			wantURL: "https://github.com",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &fakeExecutor{}
			o, err := validateOpener(Opener{Name: "t", Actions: c.actions})
			if err != nil {
				t.Fatalf("validateOpener 失败: %v", err)
			}
			svc := &Service{executor: fake, baseURL: base}
			if err := svc.Open(o, c.role, c.paths...); err != nil {
				t.Fatalf("Open 失败: %v", err)
			}
			if len(fake.calls) != 1 {
				t.Fatalf("期望 executor 被调用 1 次，实际 %d 次", len(fake.calls))
			}
			call := fake.calls[0]
			if call.bin != sysOpenBin() {
				t.Fatalf("bin 应为系统 opener %q, got %q", sysOpenBin(), call.bin)
			}
			if len(call.args) != 1 || call.args[0] != c.wantURL {
				t.Fatalf("args = %v, want [%s]", call.args, c.wantURL)
			}
		})
	}

	t.Run("parseAction 冒号后空格可选", func(t *testing.T) {
		k, tmpl, err := parseAction("url:  /workbench?path=$0")
		if err != nil || k != KindURL || tmpl != "/workbench?path=$0" {
			t.Fatalf("parseAction = (%q,%q,%v)", k, tmpl, err)
		}
		k2, tmpl2, err2 := parseAction("exec:open $0")
		if err2 != nil || k2 != KindExec || tmpl2 != "open $0" {
			t.Fatalf("parseAction = (%q,%q,%v)", k2, tmpl2, err2)
		}
	})
}
