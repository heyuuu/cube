package opener

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// ActionKind 动作类型：动作串 `<kind>:<模板>` 的前缀（首个冒号前，前后空格可选）。
type ActionKind string

const (
	// KindExec 启动子进程执行命令（sh 风格分词 + $N 占位符渲染）。
	KindExec ActionKind = "exec"
	// KindURL 打开 URL：`/` 开头为站内路由（拼 baseURL 后打开），`http(s)://` 开头为外部 URL。
	KindURL ActionKind = "url"
)

// actionSpec 单个 role 的动作模板：构造时完成前缀解析、值域与占位符校验。
type actionSpec struct {
	kind   ActionKind // exec / url
	tmpl   string     // 前缀后的模板
	tokens []string   // exec 形态的 tmpl 分词结果（url 形态为 nil）
}

// parseAction 解析动作串 `<kind>:<模板>`：只认首个冒号，冒号后空格可选；
// 前缀必须是合法 kind（无默认前缀），模板非空。
func parseAction(raw string) (ActionKind, string, error) {
	idx := strings.Index(raw, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("动作串缺少 kind 前缀（应为 exec: 或 url: 开头）: %q", raw)
	}
	kind := ActionKind(strings.TrimSpace(raw[:idx]))
	switch kind {
	case KindExec, KindURL:
	default:
		return "", "", fmt.Errorf("未知动作前缀 %q（合法值：exec/url）", raw[:idx])
	}
	tmpl := strings.TrimSpace(raw[idx+1:])
	if tmpl == "" {
		return "", "", fmt.Errorf("动作串 %q 前缀后的模板为空", kind)
	}
	return kind, tmpl, nil
}

// validateAction 校验 action 合法性，成功时返回解析后结构
func validateAction(role Role, action string) (*actionSpec, error) {
	slotCount, err := roleSlotCount(role)
	if err != nil {
		return nil, err
	}

	kind, tmpl, err := parseAction(action)
	if err != nil {
		return nil, fmt.Errorf("解析 action 动作失败: raw=`%s`, err=%w", action, err)
	}

	var tokens []string
	switch kind {
	case KindExec:
		tokens, err = tokenizeCmd(tmpl)
		if err != nil {
			return nil, fmt.Errorf("exec action tokenize 失败: raw=`%s`, err=%w", action, err)
		}
		if len(tokens) == 0 {
			return nil, fmt.Errorf("exec action 缺少参数: raw=`%s`, err=%w", action, err)
		}
	case KindURL:
		if err := validateURLTemplate(tmpl); err != nil {
			return nil, fmt.Errorf("url action 参数校验失败: raw=`%s`, err=%w", action, err)
		}
	}
	for _, n := range scanPlaceholders(tmpl) {
		if n >= slotCount {
			return nil, fmt.Errorf("action 动作占位符 $%d 越界 (槽个数=%d，合法索引 0..%d): raw=%s", n, slotCount, slotCount-1, action)
		}
	}

	return &actionSpec{kind: kind, tmpl: tmpl, tokens: tokens}, nil
}

// buildActionArgs 构造 action 参数，主要用于 runAction, 单独函数仅为了方便测试
func buildActionArgs(role Role, action string, slotArgs []string, baseURL string) (bin string, args []string, err error) {
	slotCount, _ := roleSlotCount(role)
	if len(slotArgs) != slotCount {
		return "", nil, fmt.Errorf("role %s 需要 %d 个路径参数，实际传入 %d", role, slotCount, len(slotArgs))
	}

	spec, err := validateAction(role, action)
	if err != nil {
		return "", nil, err
	}

	if spec.kind == KindURL {
		var full string
		if strings.HasPrefix(spec.tmpl, "/") {
			full = baseURL + mustRender(spec.tmpl, slotArgs, true)
		} else {
			full = mustRender(spec.tmpl, slotArgs, false)
		}
		return sysOpenBin(), []string{full}, nil
	}

	// 先对整条 cmd 渲染占位符（cmd[0] 也可能是占位符，如用路径本身作可执行文件），
	// 再取 [0] 为 bin、[1:] 为 args。
	rendered, used := renderArgs(spec.tokens, slotArgs)
	// cmd 参数里没有任何占位符时，把路径按顺序追加到末尾（兼容纯 "code" 配置）
	if used == 0 {
		rendered = append(rendered, slotArgs...)
	}
	bin = resolveBin(rendered[0])
	args = rendered[1:]
	return bin, args, nil
}

// runAction 执行 action 的主入口
func runAction(executor Executor, role Role, action string, slotArgs []string, baseURL string) error {
	bin, args, err := buildActionArgs(role, action, slotArgs, baseURL)
	if err != nil {
		return err
	}

	slog.Debug("runAction", "bin", bin, "args", args)
	return executor.Run(bin, args...)
}

// tokenizeCmd sh 风格分词：空白分隔，单/双引号内内容（含空格）为一个 token；
// 双引号内及裸词态支持反斜杠转义下一个字符。未闭合引号返回中文错误——
// 配置校验发生在保存/加载边界，此时引号未闭合即配置写错，应拦下而非运行期才失败。
func tokenizeCmd(line string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	hasToken := false
	flush := func() {
		if hasToken {
			tokens = append(tokens, cur.String())
			cur.Reset()
			hasToken = false
		}
	}

	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		switch {
		case ch == '\\' && i+1 < len(runes):
			cur.WriteRune(runes[i+1])
			hasToken = true
			i++
		case ch == '\'' || ch == '"':
			quote := ch
			hasToken = true
			closed := false
			for i++; i < len(runes); i++ {
				if runes[i] == quote {
					closed = true
					break
				}
				if quote == '"' && runes[i] == '\\' && i+1 < len(runes) {
					i++
				}
				cur.WriteRune(runes[i])
			}
			if !closed {
				return nil, fmt.Errorf("cmd 引号 %c 未闭合", quote)
			}
		case ch == ' ' || ch == '\t':
			flush()
		default:
			cur.WriteRune(ch)
			hasToken = true
		}
	}
	flush()
	return tokens, nil
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// renderTemplate 渲染模板内所有 $<数字>（索引在 paths 范围内）为对应路径，返回渲染结果
// 与是否发生替换；越界占位符原样保留。escape=true（url 站内路由）时替换值做 query encode
// （路由端 decode 还原）；escape=false（exec 分词后）照常子串替换，路径含空格/引号也不会被再次分词。
func renderTemplate(tmpl string, paths []string, escape bool) (string, bool) {
	var b strings.Builder
	used := false
	for i := 0; i < len(tmpl); i++ {
		if tmpl[i] == '$' && i+1 < len(tmpl) && isDigit(tmpl[i+1]) {
			j := i + 1
			for j < len(tmpl) && isDigit(tmpl[j]) {
				j++
			}
			n, _ := strconv.Atoi(tmpl[i+1 : j])
			if n < len(paths) {
				if escape {
					b.WriteString(url.QueryEscape(paths[n]))
				} else {
					b.WriteString(paths[n])
				}
				used = true
			} else {
				b.WriteString(tmpl[i:j])
			}
			i = j - 1
			continue
		}
		b.WriteByte(tmpl[i])
	}
	return b.String(), used
}

// mustRender url 模板渲染（不关心 used，url 动作占位符缺失时原样保留即可）。
func mustRender(tmpl string, paths []string, escape bool) string {
	s, _ := renderTemplate(tmpl, paths, escape)
	return s
}

// scanPlaceholders 扫描 token 内所有 $<数字> 占位符（数字取最长连续段），返回索引列表。
func scanPlaceholders(token string) []int {
	var indices []int
	for i := 0; i < len(token); i++ {
		if token[i] != '$' || i+1 >= len(token) || !isDigit(token[i+1]) {
			continue
		}
		j := i + 1
		for j < len(token) && isDigit(token[j]) {
			j++
		}
		n, _ := strconv.Atoi(token[i+1 : j])
		indices = append(indices, n)
		i = j - 1
	}
	return indices
}

// renderPlaceholders 把 token 内所有 $<数字>（索引在 paths 范围内）替换为对应路径，
// 返回渲染结果与是否发生替换。越界占位符原样保留。替换发生在分词之后，
// 路径含空格/引号也不会被再次分词。
func renderPlaceholders(token string, paths []string) (string, bool) {
	return renderTemplate(token, paths, false)
}

// renderArgs 渲染各 token 内的占位符 $0/$1... 为 paths 对应项（token 内子串替换，
// 支持 --wd=$0 形态；替换值不再分词，路径含空格/引号安全）。
// 返回渲染后的参数列表，以及实际命中占位符的个数。
func renderArgs(args []string, paths []string) ([]string, int) {
	out := make([]string, 0, len(args))
	used := 0
	for _, a := range args {
		rendered, hit := renderPlaceholders(a, paths)
		if hit {
			used++
		}
		out = append(out, rendered)
	}
	return out, used
}

// resolveBin 自引用替换：bin 为 "cube" 或以 "/cube" 结尾时，换成当前进程的可执行文件。
// opener 可组合 cube 自身 CLI（如 "cube ui workbench $0"），dev 环境（air /
// just run 源码直跑）PATH 里未必有 cube、或装的是旧版本；替换后始终与当前进程同源。
// os.Executable 失败时保留原值降级（交给 PATH 解析兜底）。
func resolveBin(bin string) string {
	if bin != "cube" && !strings.HasSuffix(bin, "/cube") {
		return bin
	}
	self, err := os.Executable()
	if err != nil {
		slog.Debug("解析当前可执行文件失败，opener cmd[0] 保留原值", "err", err)
		return bin
	}
	return self
}

// sysOpenBin 系统默认打开器（打开 URL / 文件）。cube 是纯 macOS 工具，恒为 open。
func sysOpenBin() string { return "open" }

// validateURLTemplate 校验 url 动作模板的值域：`/` 开头（站内路由）或 `http(s)://` 开头（外部 URL）。
func validateURLTemplate(tmpl string) error {
	if strings.HasPrefix(tmpl, "/") || strings.HasPrefix(tmpl, "http://") || strings.HasPrefix(tmpl, "https://") {
		return nil
	}
	return fmt.Errorf("url 动作模板必须以 /（站内路由）或 http(s)://（外部 URL）开头: %q", tmpl)
}
