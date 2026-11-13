package template

import (
	"fmt"
	"os"
	"os/exec"
)

// validateTarget 预检目标路径（绝对路径，cmd 层已展开）：已存在时必须是空目录（防覆盖既有内容）。
// 由 Service.CheckTarget 供 cmd 层在 clone/交互之前调用，让路径错误第一时间暴露。
func validateTarget(targetPath string) error {
	if info, err := os.Stat(targetPath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("目标路径已存在且不是目录: %s", targetPath)
		}
		entries, err := os.ReadDir(targetPath)
		if err != nil {
			return fmt.Errorf("读取目标目录失败: %w", err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("目标目录非空，拒绝覆盖: %s", targetPath)
		}
	}
	return nil
}

// runInit 逐条执行 init 命令（sh -c，cwd 为生成后的项目根，stdio 接终端），任一失败中止。
func runInit(tpl *Template, target string, vars map[string]string) error {
	cmds, err := interpolateInit(tpl.Meta().Init, vars)
	if err != nil {
		return err
	}

	for i, cmd := range cmds {
		c := exec.Command("sh", "-c", cmd)
		c.Dir = target
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		fmt.Printf("init[%d]: %s\n", i+1, cmd)
		if err := c.Run(); err != nil {
			return fmt.Errorf("init 第 %d 条命令失败（已中止后续命令）: %q: %w", i+1, cmd, err)
		}
	}
	return nil
}
