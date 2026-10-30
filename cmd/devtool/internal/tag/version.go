// Package tag 三段式版本 tag 的解析与递增计算（devtool tag 子命令使用）。
package tag

import (
	"fmt"
	"strconv"
	"strings"
)

// VersionTag 表示三段式版本号 X.Y.Z，对应 tag 名 `v{X.Y.Z}` 中的版本部分。
type VersionTag struct {
	Major, Minor, Patch int
}

// ParseVersionTag 解析三段式版本号 Tag（如 "v0.1.0"），
// 格式不合法时第二个返回值为 false。
func ParseVersionTag(s string) (VersionTag, bool) {
	var zero VersionTag

	if !strings.HasPrefix(s, "v") {
		return zero, false
	}

	parts := strings.Split(s[1:], ".")
	if len(parts) != 3 {
		return zero, false
	}

	var v VersionTag
	nums := [3]*int{&v.Major, &v.Minor, &v.Patch}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return zero, false
		}
		*nums[i] = n
	}
	return v, true
}

// String 返回版本号 tag 字符串。
func (v VersionTag) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare 按数值逐段比较：大于返回 +1，小于返回 -1，相等返回 0。
func (v VersionTag) Compare(o VersionTag) int {
	for _, d := range [3]int{v.Major - o.Major, v.Minor - o.Minor, v.Patch - o.Patch} {
		if d < 0 {
			return -1
		} else if d > 0 {
			return 1
		}
	}
	return 0
}

// NextMajor 下一个 Major 版本
func (v VersionTag) NextMajor() VersionTag {
	return VersionTag{Major: v.Major + 1, Minor: 0, Patch: 0}
}

// NextMinor 下一个 Minor 版本
func (v VersionTag) NextMinor() VersionTag {
	return VersionTag{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
}

// NextPatch 下一个 Patch 版本
func (v VersionTag) NextPatch() VersionTag {
	return VersionTag{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
}
