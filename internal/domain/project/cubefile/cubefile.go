// Package cubefile 提供项目根 `.cube/cube.json` 的文件格式层（提案 1030）。
//
// cube.json 是「项目自身的 cube 配置」的容器：进 git 跟仓库走、跨机器共享。
// 节级读写语义（节缺失哨兵错误 / 未知节保留 / 原子写）由外部包
// github.com/heyuuu/x/store 的 section API 提供，本包按节组装领域类型，
// 不 interpret 各节——各节的业务语义归各自 domain 包（如 workspace
// 规则解析在 project/workspace）。
package cubefile

import (
	"path/filepath"
)

const cubeFileName = ".cube/cube.json"

// FilePath 返回 root 的 cube.json 文件路径（供展示与提示引用；节级读写走各节 API）。
func FilePath(root string) string {
	return filepath.Join(root, cubeFileName)
}
