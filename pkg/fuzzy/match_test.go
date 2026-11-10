package fuzzy

import (
	"reflect"
	"testing"
)

// TestMatchBy_EmptyQuery 空查询返回全部（克隆）。
func TestMatchBy_EmptyQuery(t *testing.T) {
	targets := []string{"a", "b", "c"}
	got := MatchBy("", targets, func(s string) string { return s }, nil)
	if !reflect.DeepEqual(got, targets) {
		t.Fatalf("空 query 应返回全部，实际 %v", got)
	}
	// 空格 query 也算空
	got = MatchBy("   ", targets, func(s string) string { return s }, nil)
	if !reflect.DeepEqual(got, targets) {
		t.Fatalf("空格 query 应返回全部，实际 %v", got)
	}
}

func TestMatchBy_EmptyTargets(t *testing.T) {
	got := MatchBy("x", []string{}, func(s string) string { return s }, nil)
	if len(got) != 0 {
		t.Fatalf("空 targets 应返回空，实际 %v", got)
	}
}

// TestMatchBy_FilterAndSort 匹配过滤 + 按分排序。
func TestMatchBy_FilterAndSort(t *testing.T) {
	targets := []string{"github:cube", "pers:secret", "github:card"}
	// 查 "c"：应匹配 cube 和 card，secret 不含连续 c 开头也可能匹配
	got := MatchBy("cube", targets, func(s string) string { return s }, nil)
	// cube 完整匹配分数最高应排第一
	if len(got) == 0 {
		t.Fatalf("应至少匹配到 cube")
	}
	if got[0] != "github:cube" {
		t.Fatalf("cube 应排第一，实际 %v", got)
	}
}

// TestMatchBy_NilScorerUsesDefault scorer 为 nil 时用默认 scorer。
func TestMatchBy_NilScorerUsesDefault(t *testing.T) {
	targets := []string{"alpha", "beta"}
	// 不 panic 即可
	got := MatchBy("a", targets, func(s string) string { return s }, nil)
	if len(got) == 0 {
		t.Fatalf("alpha 应匹配到 a")
	}
}
