package cubefile

import (
	"os"
	"path/filepath"
	"testing"

	"cube/internal/testfixture"
)

func writeFile(t *testing.T, root, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".cube"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cube", "cube.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// 存在性语义：空数组也是显式声明（WorkspacesSet=true），字段缺失才是「未声明」。
func TestLoadPresence(t *testing.T) {
	root := testfixture.NewWorkspace(t).Dir

	writeFile(t, root, `{"workspaces":[]}`)
	f, ok := Load(root)
	if !ok || !f.WorkspacesSet || len(f.Workspaces) != 0 {
		t.Fatalf("空数组应是显式声明: ok=%v %+v", ok, f)
	}

	writeFile(t, root, `{"workspaceScanRule":"pnpm"}`)
	f, ok = Load(root)
	if !ok || f.WorkspacesSet || f.WorkspaceScanRule != "pnpm" {
		t.Fatalf("字段缺失应 WorkspacesSet=false: ok=%v %+v", ok, f)
	}

	// 坏 JSON / 文件不存在 → (nil,false)，降级语义由调用方决定
	writeFile(t, root, `{bad`)
	if _, ok := Load(root); ok {
		t.Fatal("坏 JSON 应返回 false")
	}
	if _, ok := Load(root + "-absent"); ok {
		t.Fatal("文件不存在应返回 false")
	}
}
