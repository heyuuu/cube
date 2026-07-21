package git

// 本文件封装「系统 git 命令」的调用，通过 exec.Command 启动 git 子进程完成操作。
//
// 设计原则：本包内将 git 操作按「读 / 写」拆分到两个文件：
//   - git.go   —— 系统 git 命令封装。主要承载「写操作 / 需要透传输出的操作」。
//                 子进程能复用用户本地的 git 配置（凭据、SSH agent、hooks 等），
//                 并能直接把 stdout/stderr 透传给当前终端，体验明显优于纯 Go 实现。
//                 典型场景：clone（需要交互式进度、SSH 凭据）。
//   - gogit.go  —— go-git 纯 Go 实现。承载「读操作」。
//                 读 repoUrl / branches / ahead-behind / status 这类高频、无副作用、
//                 不需要凭据的操作时，起子进程的 fork/exec 开销会成为性能瓶颈
//                 （典型场景：scan 出几十上百个项目，每个项目要起 3 次 git 子进程）。
//                 go-git 直接读 .git 目录，零子进程，配合缓存层把采集从 3N 次子进程
//                 降为零。
//
// 两个文件对外都暴露为 git 包；调用方按功能挑选即可，不需要关心底层实现。

import (
	"os/exec"
	"strings"
)

// gitCmdRun 在指定 path 下执行一次 git 命令，返回 stdout 内容。
// 仅作为本包内部「写操作」类封装的共用底座；读操作请使用 gogit.go 中的封装。
func gitCmdRun(path string, command string, args ...string) (string, error) {
	realArgs := append([]string{"-C", path, command}, args...)
	cmd := exec.Command("git", realArgs...)

	var buf strings.Builder
	cmd.Stdout = &buf
	err := cmd.Run()

	return buf.String(), err
}
