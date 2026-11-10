// Package workspace 提供 monorepo workspace 的规则解析（提案 1030）。
//
// 规则来源 `.cube/cube.json` 的 workspaces 节（文件格式层见 project/cubefile）：
// 每条 {name?, path}——path 为相对根的精确目录 / glob 通配 / 特殊规则
// （<npm>/<pnpm> 读标准 monorepo 声明文件）；节缺失时用程序默认规则。
// 本包只做解析与推导——持久化的是规则本身，推导结果由 snapshot 采集侧带入快照。
package workspace

// Workspace 一个可作为打开入口的子目录（规则解析推导而来）。
// Path 相对所属根（主项目根或 worktree 根），已清理。
type Workspace struct {
	Name string `json:"name"` // 显示名
	Path string `json:"path"` // 相对所属根的路径
}
