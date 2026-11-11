package opener

import (
	"testing"
)

// ---------- validateOpener actions 校验 ----------

func TestValidateOpenerActions(t *testing.T) {
	cases := []struct {
		name    string
		actions map[Role]string
		wantErr bool
	}{
		{"actions 缺失报错", nil, true},
		{"actions 空报错", map[Role]string{}, true},
		{"无 kind 前缀报错", map[Role]string{RoleOpenDir: "code $0"}, true},
		{"未知 kind 报错", map[Role]string{RoleOpenDir: "shell: code $0"}, true},
		{"前缀后模板为空报错", map[Role]string{RoleOpenDir: "exec:"}, true},
		{"url 非法值域报错", map[Role]string{RoleOpenDir: "url: ftp://x"}, true},
		{"url 占位符越界报错", map[Role]string{RoleDiffFile: "url: /x?a=$2"}, true},
		{"外部 URL 合法", map[Role]string{RoleOpenDir: "url: https://github.com"}, false},
		{"未知 role 报错", map[Role]string{"unknown-role": "code"}, true},
		{"单条 cmd 缺失报错", map[Role]string{RoleOpenDir: ""}, true},
		{"单条 cmd 纯空白报错", map[Role]string{RoleOpenDir: "   "}, true},
		{"cmd 引号未闭合报错", map[Role]string{RoleOpenDir: `exec: code "abc`}, true},
		{"仅可执行文件(无占位符)合法", map[Role]string{RoleOpenDir: "exec: code"}, false},
		{"占位符不越本 role 槽数合法", map[Role]string{RoleDiffFile: "exec: bcompare $0 $1"}, false},
		{"占位符越本 role 槽数报错", map[Role]string{RoleOpenDir: "exec: bcompare $0 $1"}, true},
		{"token 内嵌占位符同样校验越界", map[Role]string{RoleOpenDir: "exec: tool --wd=$1"}, true},
		{"不同 role 槽数不同可并存（无一致性约束）", map[Role]string{RoleOpenDir: "exec: code $0", RoleDiffFile: "exec: code --diff $0 $1"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := Opener{Name: "test", Actions: c.actions}
			o, err := validateOpener(o)
			if c.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际 o=%+v err=nil", o)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外报错: %v", err)
			}
		})
	}
}

func TestTitleDefault(t *testing.T) {
	o1, _ := validateOpener(Opener{Name: "code", Actions: map[Role]string{"open-dir": `exec: code`}})
	if got := o1.Title; got != "用 code 打开" {
		t.Fatalf("缺省 title 应由 name 生成, got %q", got)
	}
	o2, _ := validateOpener(Opener{Name: "finder", Title: "打开所在目录", Actions: map[Role]string{"open-dir": `exec: open $0`}})
	if got := o2.Title; got != "打开所在目录" {
		t.Fatalf("配置 title 应原样生效, got %q", got)
	}
}
