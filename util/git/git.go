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
	"log/slog"
	"os"
	"os/exec"
	"strconv"
)

// Clone 使用系统 git 克隆仓库到 localPath，stdout/stderr 透传给当前终端。
//
// 为什么不用 go-git：
//   - clone 需要交互式进度输出（传输速率、剩余时间）
//   - 需要 SSH 凭据 / HTTPS 凭据助手 / git-credential-osxkeychain 等本地生态
//   - 需要 respect 用户 ~/.gitconfig 的 hooks、alias、protocol 配置
//   系统 git 都能原生复用，go-git 在这些场景下兼容性差、体验差。
//
// 参数：
//   - localPath: 克隆目标目录（绝对路径）
//   - repoUrl:   仓库地址（SSH 或 HTTPS）
//   - depth:     克隆深度，<=0 表示不限制（完整克隆）
//   - branch:    指定分支名，空串表示克隆默认分支
func Clone(localPath string, repoUrl string, depth int, branch string) error {
	args := []string{"clone", repoUrl, localPath}
	if depth > 0 {
		args = append(args, "--depth="+strconv.Itoa(depth))
	}
	if branch != "" {
		args = append(args, "--branch="+branch)
	}

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Info("Run cmd", "cmd", cmd.String())
	return cmd.Run()
}
