package git

// ref 查询：本地与远程分支 / tag（本地 refs 与远端 ls-remote 实查）/ 默认分支 /
// HEAD 与父提交 / ahead-behind。remote 配置查询（remote -v）在 remote.go。
// 执行核与错误约定见 run.go。

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ref namespace 前缀：git 规范全名的三棵主子树（本包多处与 workbench/cmd 的
// 拼装/前缀判定共用；refs/ 树本身开放，不止这三棵）
const (
	RefHeadsPrefix   = "refs/heads/"
	RefRemotesPrefix = "refs/remotes/"
	RefTagsPrefix    = "refs/tags/"
)

// Ref 一条 git ref（本地分支 / 远端分支 / tag）的解析结果。
type Ref struct {
	Name      string `json:"name"`      // Ref Name，refs/{heads,remotes,tags}/xxx
	ShortName string `json:"shortName"` // Short Name，省略 refs/xxx/ 前缀；注意 remote 的 ShortName 含 remote 名
	Remote    string `json:"remote"`    // Remote，仅 refs/remotes/* 有效，远端 remote 名 (例如 origin)
	Branch    string `json:"branch"`    // 分支名，在 refs/heads/* 及 refs/remotes/* 时有效，分支名 (例如 master)

	// 以下 upstream 跟踪信息仅本地分支（refs/heads/*）可能非零，由 Refs 的
	// for-each-ref 单进程顺带取回（免去调用方逐分支再跑 rev-list）。
	// Ahead/Behind 基于本地已有的跟踪 ref 比对（不 fetch），口径同 AheadBehindRemote
	// ——未 fetch 过的数据可能不准。
	Upstream string `json:"upstream,omitempty"` // 上游跟踪 ref 全名（refs/remotes/<remote>/<branch>）；空 = 未配置 upstream
	Ahead    int    `json:"ahead,omitempty"`    // 相对 upstream 领先 commit 数（待推送）；未配置或 Gone 时为 0
	Behind   int    `json:"behind,omitempty"`   // 相对 upstream 落后 commit 数（待拉取）；未配置或 Gone 时为 0
	Gone     bool   `json:"gone,omitempty"`     // upstream 已配置但跟踪 ref 已不存在（远端分支被删）；此时 Ahead/Behind 无意义
}

// IsLocal 是本地分支（refs/heads/*）。
func (r Ref) IsLocal() bool {
	return r.Remote == "" && r.Branch != ""
}

// IsRemote 是远端分支（refs/remotes/*）。
func (r Ref) IsRemote() bool {
	return r.Remote != "" && r.Branch != ""
}

// IsTag 是 tag（无 branch 段）。
func (r Ref) IsTag() bool {
	return r.Branch == ""
}

// buildRef 把规范全名解析为 Ref 值对象（分组判定依赖其错误口径做跳过）。
func buildRef(refName string) (Ref, error) {
	refName = strings.TrimSpace(refName)
	switch {
	case strings.HasPrefix(refName, RefHeadsPrefix):
		shortName := strings.TrimPrefix(refName, RefHeadsPrefix)
		return Ref{Name: refName, ShortName: shortName, Branch: shortName}, nil
	case strings.HasPrefix(refName, RefTagsPrefix):
		shortName := strings.TrimPrefix(refName, RefTagsPrefix)
		return Ref{Name: refName, ShortName: shortName}, nil
	case strings.HasPrefix(refName, RefRemotesPrefix):
		shortName := strings.TrimPrefix(refName, RefRemotesPrefix)

		// 拆分 remote 名 和 branch 名
		idx := strings.Index(shortName, "/")
		if idx <= 0 {
			break
		}
		remote, branch := shortName[:idx], shortName[idx+1:]

		// refs/remotes/<remote>/HEAD 是远端默认分支的符号指针而非真实分支，跳过
		if branch == "HEAD" {
			break
		}

		return Ref{
			Name:      refName,
			ShortName: shortName,
			Remote:    remote,
			Branch:    branch,
		}, nil
	}

	// 没匹配上的情况
	return Ref{}, errors.New("暂不支持的 ref 值: " + refName)
}

// RefsResult 全量 ref 清单（规范全名，按 namespace 分组）。
type RefsResult struct {
	Locals  []Ref `json:"locals"`
	Remotes []Ref `json:"remotes"`
	Tags    []Ref `json:"tags"`
}

// refLineSep for-each-ref 输出行里各字段的分隔符。ref 名与 upstream ref 名按
// check-ref-format 规则不含空格，" | " 不会与字段内容产生歧义。
const refLineSep = " | "

// BuildRefs 构建 Refs 结果，暴露以供构造测试数据。入参是 for-each-ref 的输出行
// （"refname | upstream | track"，同 Refs 的 format）；裸 refname 行同样接受
// （等价于无 upstream 信息），测试给名字列表即可。无法解析的行跳过。
func BuildRefs(lines []string) *RefsResult {
	result := &RefsResult{}
	for _, line := range lines {
		ref, ok := parseRefLine(line)
		if !ok {
			continue
		}
		if ref.IsLocal() {
			result.Locals = append(result.Locals, ref)
		} else if ref.IsRemote() {
			result.Remotes = append(result.Remotes, ref)
		} else if ref.IsTag() {
			result.Tags = append(result.Tags, ref)
		}
	}
	return result
}

// parseRefLine 解析 Refs 输出的一行 "refname | upstream | track"，
// 裸 refname 行同样接受（等价于无 upstream 信息）。无法解析返回 false，调用方跳过。
func parseRefLine(line string) (Ref, bool) {
	parts := strings.SplitN(line, refLineSep, 3)
	ref, err := buildRef(strings.TrimSpace(parts[0]))
	if err != nil {
		return Ref{}, false
	}
	if len(parts) > 1 {
		ref.Upstream = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		ref.Ahead, ref.Behind, ref.Gone = parseUpstreamTrack(parts[2])
	}
	return ref, true
}

// parseUpstreamTrack 解析 %(upstream:track) 的输出（git 默认带方括号）：
// ""（已同步或未配置 upstream）/ "[ahead N]" / "[behind N]" / "[ahead N, behind M]" /
// "[gone]"（upstream 已配置但跟踪 ref 已不存在——远端分支被删）。
// 无法识别的内容降级为零值——track 是附加信息，不应让整个 Refs 失败。
func parseUpstreamTrack(s string) (ahead, behind int, gone bool) {
	s = strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "[]"))
	if s == "gone" {
		return 0, 0, true
	}
	for _, part := range strings.Split(s, ",") {
		kind, num, ok := strings.Cut(strings.TrimSpace(part), " ")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			continue
		}
		switch kind {
		case "ahead":
			ahead = n
		case "behind":
			behind = n
		}
	}
	return ahead, behind, false
}

// Refs 一次 for-each-ref 拉全 refs/ 树，按 namespace 前缀分组返回规范全名；
// 本地分支顺带带上 upstream 跟踪信息（Upstream / Ahead / Behind / Gone）——
// ahead/behind 计数由 git 在同一进程内完成，调用方不必再逐分支跑 rev-list。
// 只回答「仓库有哪些 ref」；「当前检出在哪个 ref」是 HEAD 状态，用 HeadRef。
// 规范全名的拼装收敛在此（workbench 的 TreeSource ref id 直接透传，不在调用方拼前缀）；
// refs/stash 等其他子树不分组不返回（现有消费方只消费三类）。
// 非仓库返回零值，不视为错误。
func Refs(path string) (*RefsResult, error) {
	if !isGitRepo(path) {
		return &RefsResult{}, nil
	}

	format := "--format=%(refname)" + refLineSep + "%(upstream)" + refLineSep + "%(upstream:track)"
	out, err := runOut(path, "for-each-ref", format)
	if err != nil {
		return nil, err
	}

	// 只剥行结束符、不做 TrimSpace：空字段时每行以尾随 " | " 结尾，
	// 整体 TrimSpace 会把末行的分隔符一起咬掉，造成末行字段错位
	result := BuildRefs(strings.Split(strings.TrimRight(out, "\n"), "\n"))
	return result, nil
}

// RemoteTag 远端仓库的一条 tag（RemoteTags 的 ls-remote 实时查询结果）。
type RemoteTag struct {
	Name   string `json:"name"`   // tag 短名（refs/tags/ 前缀已剥）
	Sha    string `json:"sha"`    // ref 直接指向的对象（annotated tag 为 tag 对象本身）
	Commit string `json:"commit"` // 剥壳后的 commit；轻量 tag 与 Sha 相同
}

// RemoteTags 实时查询 remote 端的 tag 清单（git ls-remote --tags <remote>，走网络）。
// 与 Refs().Tags 是两套口径：Refs 读本地 refs/tags/*（上次 fetch 留下的副本），
// 本函数问远端「现在」有什么——可发现远端独有、本地未 fetch 的 tag；本地独有
// tag 反之不会出现在结果里。annotated tag 的双行输出（tag 对象 + ^{} 剥壳
// commit）合并为一条。结果按名字排序。非仓库返回 (nil, nil)；remote 名不存在、
// 网络失败等报真实错误，由调用方决定降级。
func RemoteTags(path string, remote string) ([]RemoteTag, error) {
	if !isGitRepo(path) {
		return nil, nil
	}
	out, err := runOut(path, "ls-remote", "--tags", remote)
	if err != nil {
		return nil, err
	}
	tags := parseLsRemoteTags(out)
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	return tags, nil
}

// lsRemotePeelSuffix annotated tag 剥壳行（refs/tags/x^{}）的后缀，指向 commit。
const lsRemotePeelSuffix = "^{}"

// parseLsRemoteTags 解析 `git ls-remote --tags` 输出，每行 "<sha>\trefs/tags/<name>"；
// annotated tag 额外带剥壳行 "<commit>\trefs/tags/<name>^{}"，回填进同名条目的
// Commit 字段。无法解析的行跳过。
func parseLsRemoteTags(out string) []RemoteTag {
	index := make(map[string]int) // tag 名 → tags 下标，剥壳行回填 Commit 用
	var tags []RemoteTag
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		sha, refname, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		peeled := strings.HasSuffix(refname, lsRemotePeelSuffix)
		name := strings.TrimPrefix(strings.TrimSuffix(refname, lsRemotePeelSuffix), RefTagsPrefix)
		if peeled {
			if i, ok := index[name]; ok {
				tags[i].Commit = sha
			}
			continue
		}
		index[name] = len(tags)
		tags = append(tags, RemoteTag{Name: name, Sha: sha, Commit: sha})
	}
	return tags
}

// HeadRef 返回 HEAD 指向的 ref 全名。正常为 refs/heads/*；git 只强制 HEAD 在
// refs/ 内，病态挂载（symbolic-ref 指向 tag 等）时原样返回、不做 heads 假设。
// detached HEAD、非仓库返回空串——这是业务空值而非错误（Branches 的 current
// 短名与 workbench 的当前分支全名都源于此，只差在是否剥前缀）。
func HeadRef(path string) string {
	if !isGitRepo(path) {
		return ""
	}
	// symbolic-ref 失败（非 0 退出）= detached HEAD，降级空串
	out, err := runOut(path, "symbolic-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// CurrentBranch 当前检出分支短名（HEAD 目标 ref 剥 refs/heads/ 前缀）。
// detached、病态挂载到 heads 外、非仓库均返回空——与 git branch --show-current
// 口径一致（git 只强制 HEAD 在 refs/ 内，不强制在 heads 下）。
func CurrentBranch(path string) string {
	ref := HeadRef(path)
	if strings.HasPrefix(ref, RefHeadsPrefix) {
		return strings.TrimPrefix(ref, RefHeadsPrefix)
	}
	return ""
}

// DefaultBranch 返回 path 处仓库的默认分支短名（"master" / "main" 等）。
// 用于 ahead/behind 比较「主分支本地 vs 主分支远程」。
//
// 判断顺序（可靠度递减）：
//  1. origin/HEAD 指向的分支（远程仓库声明的 default branch；克隆时自动建立）
//  2. 本地 master 分支存在 → master（兼容老仓库的常见情况）
//  3. 本地 main 分支存在 → main（新仓库的常见情况）
//  4. 都没有 → 返回空串（调用方应跳过 ahead/behind）
func DefaultBranch(path string) (string, error) {
	if !isGitRepo(path) {
		return "", nil
	}
	// origin/HEAD 未设置时 symbolic-ref 报错（非 0 退出），走 fallback。
	// 不带 --short 取全名（形如 refs/remotes/origin/master），buildRef 才能拆出分支名
	if out, err := runOut(path, "symbolic-ref", RefRemotesPrefix+"origin/HEAD"); err == nil {
		if ref, err := buildRef(strings.TrimSpace(out)); err == nil {
			return ref.Branch, nil
		}
	}
	for _, candidate := range []string{"master", "main"} {
		// show-ref --verify --quiet 只用退出码表达存在性（0=存在）
		if _, err := runOut(path, "show-ref", "--verify", "--quiet", RefHeadsPrefix+candidate); err == nil {
			return candidate, nil
		}
	}
	return "", nil
}

// BranchAdd 在 path 仓库创建本地分支（`git branch`，不检出、不切 HEAD——「建分支
// 并切过去」的场景由 WorktreeAdd 覆盖）。commitish 为基点（commit/分支/tag，空 = HEAD）。
// 常见拒绝（重名 / 非法名 / 基点不存在）翻译为中文上抛；stderr 在 runOutRaw 的
// C locale 注入下稳定为英文。
func BranchAdd(path string, branch string, commitish string) error {
	args := []string{"branch", branch}
	if commitish != "" {
		args = append(args, commitish)
	}
	_, stderr, err := runOutRaw(path, args...)
	if err != nil {
		// 匹配一律小写化：git 各版本对 "Not a valid object name" 的大小写不一致
		lower := strings.ToLower(stderr)
		switch {
		case strings.Contains(lower, "already exists"):
			return fmt.Errorf("分支 %s 已存在", branch)
		case strings.Contains(lower, "not a valid branch name"):
			return errors.New(branch + " 不是合法的分支名")
		case strings.Contains(lower, "not a valid object name"):
			return errors.New("基点不存在: " + commitish)
		}
		return fmt.Errorf("git branch 执行失败: %w", err)
	}
	return nil
}

// BranchDelete 删除 path 仓库的本地分支。force 决定语义：
//   - false：git 原生 -d（分支已合并到上游或 HEAD 才允许）；
//   - true：-D 强删（丢弃未合并提交）。
//
// 「分支被任一工作副本检出」是硬约束（-D 也删不掉），由 workbench Service 基于
// WorktreeList 预检并指明检出位置；此处只把 git 的英文 stderr 拒绝原因翻译成中文
// 上抛（stderr 在 runOutRaw 的 C locale 注入下稳定为英文）。
func BranchDelete(path string, branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, stderr, err := runOutRaw(path, "branch", flag, branch)
	if err != nil {
		if strings.Contains(stderr, "not fully merged") {
			return errors.New("分支 " + branch + " 尚未合并，非 force 删除被拒绝（可勾选 force 后重试）")
		}
		if strings.Contains(stderr, "checked out") {
			return errors.New("分支 " + branch + " 正被检出，无法删除")
		}
		return fmt.Errorf("git branch "+flag+" 执行失败: %w", err)
	}
	return nil
}

// Reset 把 path 仓库的 HEAD 重置到 target（分支名/commit/tag）。hard 决定语义：
//   - false：默认 mixed——移动 HEAD 并重置暂存区，工作区改动保留；
//   - true：--hard——暂存区与工作区改动一并丢弃，不可恢复。
//     注意：reset --hard 不碰未跟踪文件，需要一并清掉新增文件时随后调用 Clean。
//
// 不做「脏工作区」预检：hard 丢弃改动是调用方（前端弹窗）显式确认过的选择。
// target 不存在等拒绝翻译成中文上抛（stderr 在 runOutRaw 的 C locale 注入下稳定为英文）。
func Reset(path string, target string, hard bool) error {
	args := []string{"reset", target}
	if hard {
		args = append(args, "--hard")
	}
	_, stderr, err := runOutRaw(path, args...)
	if err != nil {
		lower := strings.ToLower(stderr)
		if strings.Contains(lower, "unknown revision") || strings.Contains(lower, "ambiguous argument") {
			return errors.New("重置目标不存在: " + target)
		}
		return fmt.Errorf("git reset 执行失败: %w", err)
	}
	return nil
}

// Clean 删除 path 仓库的未跟踪文件与目录（git clean -fd，不碰 .gitignore 忽略的文件）。
// 典型组合：Reset --hard 之后调用它清掉残留的新增文件——reset 不碰未跟踪文件。
func Clean(path string) error {
	if _, _, err := runOutRaw(path, "clean", "-fd"); err != nil {
		return fmt.Errorf("git clean 执行失败: %w", err)
	}
	return nil
}

// HeadSha 返回 path 仓库 HEAD 指向 commit 的完整 sha（detached HEAD 同样适用）。
// 空仓库（尚无任何 commit）时 rev-parse 报错，原样返回错误。
// 只需 sha 时不要改用 LoadRepoStatus 的 Sha 字段——status 要扫描工作区，
// rev-parse 是 O(1)。
func HeadSha(path string) (string, error) {
	if !isGitRepo(path) {
		return "", nil
	}
	out, err := runOut(path, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return firstLine(out), nil
}

// ParentSha 返回 ref 的父提交完整 sha。
// 根提交没有父、ref 不存在时 git 报错并原样返回——是否降级（如改用空树）
// 由调用方决定，见 workbench.Changes。
func ParentSha(path string, ref string) (string, error) {
	out, err := runOut(path, "rev-parse", ref+"^")
	if err != nil {
		return "", err
	}
	return firstLine(out), nil
}

// AheadBehindRemote 计算本地分支 localBranch 相对指定 remote 的 remoteBranch 的
// 领先 / 落后数。
//   - ahead  = 本地有、远程没有的 commit 数（待推送）
//   - behind = 远程有、本地没有的 commit 数（待拉取）
//
// 比较基准显式拆成 remote 名 + 分支名，便于比较非 origin 的远程分支；
// snapshot 用它做「默认分支 vs origin/<默认分支>」的列表页 ahead/behind。
// 任一 ref 缺失（如该 remote 没有这个分支）返回 (0, 0, nil)，不视为错误。
// 基于本地已有 commit 比对（不 fetch），未 fetch 过的数据可能不准——
// 与「缓存场景接受 stale」的整体策略一致。
func AheadBehindRemote(path string, localBranch, remoteName, remoteBranch string) (ahead, behind int, err error) {
	return aheadBehindRefs(path, RefHeadsPrefix+localBranch, RefRemotesPrefix+remoteName+"/"+remoteBranch)
}

// aheadBehindRefs 用 `rev-list --left-right --count left...right` 计算双方各自
// 独有的 commit 数（left-only = ahead，right-only = behind）。
// rev-list 对缺失的 ref 报错退出，此时降级为零值不报错（缺 ref 属业务空值场景）。
func aheadBehindRefs(path, leftRef, rightRef string) (ahead, behind int, err error) {
	if !isGitRepo(path) {
		return 0, 0, nil
	}
	out, err := runOut(path, "rev-list", "--left-right", "--count", leftRef+"..."+rightRef)
	if err != nil {
		return 0, 0, nil
	}
	ahead, behind, ok := parseCountPair(out)
	if !ok {
		return 0, 0, nil
	}
	return ahead, behind, nil
}

// parseCountPair 解析 rev-list --count 的 "ahead\tbehind" 输出。
func parseCountPair(out string) (int, int, bool) {
	left, right, ok := strings.Cut(strings.TrimSpace(out), "\t")
	if !ok {
		return 0, 0, false
	}
	a, errA := strconv.Atoi(left)
	b, errB := strconv.Atoi(right)
	if errA != nil || errB != nil {
		return 0, 0, false
	}
	return a, b, true
}
