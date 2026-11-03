// command 子命令：生成 cobra 命令脚手架——根命令（internal/cmd/<name>.go）
// 或命令组子命令（internal/cmd/<group>/<name>.go，组未初始化时一并建组入口 cmd.go）。

package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// internal/cmd/root.go 与组入口 cmd.go 的注入锚点（与源文件内标记保持一致，改一侧须同步另一侧）
const (
	subCmdMarker = "/* <<subcommand inject point>> */"
	rootCmdFile  = "internal/cmd/root.go"
)

// genCommand 生成命令脚手架并注入注册点：
//   - 无 group：建 internal/cmd/<name>.go，在 root.go 注册
//   - 有 group：建 internal/cmd/<group>/<name>.go；组入口 cmd.go 不存在则创建并在 root.go 注册，
//     存在则在组入口 NewCmd 里追加 AddCommand
func genCommand(module string, args []string) error {
	var group, name string
	if len(args) == 1 {
		name = args[0]
	} else {
		group, name = args[0], args[1]
	}
	exported, err := checkDomain(name)
	if err != nil {
		return err
	}
	if group != "" {
		if _, err := checkDomain(group); err != nil {
			return err
		}
	}

	data := map[string]string{
		"module":      module,
		"group":       group,
		"name":        name,
		"exported":    exported,
		"subName":     name,
		"subExported": exported,
	}

	// root.go 先于建文件检查：执行目录不对时避免留下半成品
	if _, err := os.Stat(rootCmdFile); err != nil {
		return fmt.Errorf("找不到 %s（请在工具目录下执行）: %w", rootCmdFile, err)
	}

	if group == "" {
		return genRootCommand(data)
	}
	return genGroupCommand(data)
}

// genRootCommand 生成根命令文件并注入 root.go。
func genRootCommand(data map[string]string) error {
	target := "internal/cmd/" + data["name"] + ".go"
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("目标文件已存在，拒绝覆盖: %s", target)
	}

	src, err := renderTemplate("cmd_root.gotmpl", data)
	if err != nil {
		return err
	}
	if err := writeFormatted(target, src); err != nil {
		return err
	}
	fmt.Println("已创建", target)

	line := fmt.Sprintf("cmd.AddCommand(new%sCmd(a)) // %s\n", data["exported"], data["name"])
	if err := injectRootCmd(rootCmdFile, line, ""); err != nil {
		return err
	}
	fmt.Printf("已注入 root.go: new%sCmd\n", data["exported"])
	return nil
}

// genGroupCommand 生成命令组子命令；组入口不存在则创建并在 root.go 注册。
func genGroupCommand(data map[string]string) error {
	group := data["group"]
	groupDir := filepath.Join("internal/cmd", group)
	groupCmdFile := filepath.Join(groupDir, "cmd.go")
	target := filepath.Join(groupDir, data["name"]+".go")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("目标文件已存在，拒绝覆盖: %s", target)
	}

	if _, err := os.Stat(groupCmdFile); os.IsNotExist(err) {
		// 组未初始化：建组入口（首个子命令直接写进模板），并在 root.go 注册组
		if err := os.MkdirAll(groupDir, 0o755); err != nil {
			return err
		}
		src, err := renderTemplate("cmd_group.gotmpl", data)
		if err != nil {
			return err
		}
		if err := writeFormatted(groupCmdFile, src); err != nil {
			return err
		}
		fmt.Println("已创建", groupCmdFile)

		line := fmt.Sprintf("cmd.AddCommand(%s.NewCmd(a)) // %s 相关\n", group, group)
		importPath := data["module"] + "/internal/cmd/" + group
		if err := injectRootCmd(rootCmdFile, line, importPath); err != nil {
			return err
		}
		fmt.Printf("已注入 root.go: %s.NewCmd\n", group)
	} else if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", groupCmdFile, err)
	} else {
		// 组已存在：在组入口 NewCmd 追加注册行
		if err := injectGroupCmd(groupCmdFile, data); err != nil {
			return err
		}
	}

	src, err := renderTemplate("cmd_sub.gotmpl", data)
	if err != nil {
		return err
	}
	if err := writeFormatted(target, src); err != nil {
		return err
	}
	fmt.Println("已创建", target)
	return nil
}

// injectRootCmd 在 root.go 的注入锚点处追加注册行；importPath 非空时同步追加 import。
func injectRootCmd(path, line, importPath string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	text := string(src)

	if strings.Contains(text, strings.TrimSpace(line)) {
		return fmt.Errorf("root.go 已包含注册行 %q，拒绝重复注入", strings.TrimSpace(line))
	}
	if !strings.Contains(text, subCmdMarker) {
		return fmt.Errorf("root.go 缺少锚点 %q，注入中止（检查标记是否被改动）", subCmdMarker)
	}

	if importPath != "" {
		if !strings.Contains(text, `"`+importPath+`"`) {
			text = strings.Replace(text, "\n\t\"cube/internal/app\"\n",
				"\n\t\"cube/internal/app\"\n\t\""+importPath+"\"\n", 1)
		}
	}
	text = strings.Replace(text, subCmdMarker, line+"\n"+subCmdMarker, 1)
	return writeFormatted(path, text)
}

// injectGroupCmd 在组入口 cmd.go 的 NewCmd 里追加注册行。
func injectGroupCmd(path string, data map[string]string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	text := string(src)

	line := fmt.Sprintf("cmd.AddCommand(new%sCmd(a)) // %s", data["exported"], data["name"])
	if strings.Contains(text, "new"+data["exported"]+"Cmd(") {
		return fmt.Errorf("%s 已包含 new%sCmd，拒绝重复生成", path, data["exported"])
	}

	if strings.Contains(text, subCmdMarker) {
		text = strings.Replace(text, subCmdMarker, line+"\n\n"+subCmdMarker, 1)
	} else {
		// 锚点缺失（如手工建的组入口）时退化为在 NewCmd 的 return cmd 前插入
		newCmdAt := strings.Index(text, "func NewCmd(")
		if newCmdAt < 0 {
			return fmt.Errorf("%s 缺少 NewCmd 函数，注入中止", path)
		}
		retIdx := strings.Index(text[newCmdAt:], "\n\treturn cmd")
		if retIdx < 0 {
			return fmt.Errorf("%s 缺少锚点 %q 且找不到 return cmd，注入中止", path, subCmdMarker)
		}
		insertAt := newCmdAt + retIdx
		text = text[:insertAt] + line + "\n" + text[insertAt:]
	}
	return writeFormatted(path, text)
}
