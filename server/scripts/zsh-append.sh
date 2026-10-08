
# ---- cube shell 扩展（由 make install 追加，勿手改；source 本文件后可用） ----

# p <args>：以本目录模式跑 cube（等价 cube <args> --local，query 缺省以 cwd 定位项目），
# 交互手敲的形态；脚本/Alfred 等非 shell 上下文仍直接用 cube --local
p() {
	cube "$@" --local
}

# pz <query>：模糊选项目并 cd 到其打开目标（主目录/worktree/workspace）
# cube path 的 TUI 画在 /dev/tty，stdout 只有干净的一行路径，可安全被 $( ) 捕获；
# 取消/非 TTY 多目标时 cube 以非零码退出，此处直接返回不 cd
pz() {
	local dir
	dir=$(cube path "$@") || return
	[[ -n $dir ]] && cd -- "$dir"
}

# ---- p / pz 的 TAB 补全：转发给 cube 的 cobra 补全（__complete 协议），不手写候选 ----
# 注意：本文件由 make install 拼接在 cobra 生成的 completion 之后，
# 上文已有 `_cube`（cube 的 compdef 函数）可直接复用

# p 即 cube：把命令名改写回 cube 后委托上文 cobra 生成的 _cube。
# --local 只改 query 缺省时的定位行为，不影响候选内容，补全期无需注入
_cube_p() {
	words=(cube "${(@)words[2,-1]}")
	_cube
}

# pz 即 `cube path <args>`。path 的 query 没有动态候选（cmd 包无 ValidArgsFunction），
# 协议 directive 为 Default 时也不回落文件名补全——query 是项目模糊检索词，文件候选全是噪音
_cube_pz() {
	local out line directive tab=$'\t'
	local -a lines completions
	out=$(cube __complete path "${(@)words[2,CURRENT-1]}" "${words[CURRENT]}" 2>/dev/null)
	lines=("${(@f)out}")
	directive=0
	if [[ ${lines[-1]} == :* ]]; then
		directive=${lines[-1]#:}
		lines=("${(@)lines[1,-2]}")
	fi
	(( directive & 1 )) && return  # ShellCompDirectiveError：静默收场
	for line in "${lines[@]}"; do
		[[ -n $line ]] || continue
		line=${line//:/\\:}
		completions+=(${line//$tab/:})
	done
	(( ${#completions} )) && _describe -t pz 'cube path' completions
}

# compdef 由 compinit 提供；非交互 source（无补全系统）时跳过注册
(( $+functions[compdef] )) && compdef _cube_p p
(( $+functions[compdef] )) && compdef _cube_pz pz
