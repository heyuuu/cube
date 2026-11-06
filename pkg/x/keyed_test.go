package x

import (
	"testing"
)

func TestUpsertKeyed(t *testing.T) {
	type item struct {
		k string
		v int
	}
	keyOf := func(x item) string { return x.k }
	list := []item{{"a", 1}, {"b", 2}}

	got := UpsertKeyed(list, keyOf, item{"b", 9})
	if len(got) != 2 || got[1].v != 9 {
		t.Fatalf("命中键应原位替换: %+v", got)
	}
	got = UpsertKeyed(list, keyOf, item{"c", 3})
	if len(got) != 3 || got[2].k != "c" {
		t.Fatalf("未命中键应追加末尾: %+v", got)
	}
}

func TestRemoveKeyed(t *testing.T) {
	type item struct{ k string }
	keyOf := func(x item) string { return x.k }
	list := []item{{"a"}, {"b"}, {"a"}}

	rest, removed := RemoveKeyed(list, keyOf, "a")
	if !removed || len(rest) != 1 || rest[0].k != "b" {
		t.Fatalf("应删除全部命中项: rest=%+v removed=%v", rest, removed)
	}
	rest, removed = RemoveKeyed(list, keyOf, "zz")
	if removed || len(rest) != len(list) {
		t.Fatalf("未命中应原样返回且 removed=false: %+v %v", rest, removed)
	}
}

func TestDuplicateKeyed(t *testing.T) {
	type item struct{ a, b string }
	keyOf := func(x item) [2]string { return [2]string{x.a, x.b} }

	if _, ok := DuplicateKeyed([]item{{"h", "/x"}, {"h", "/y"}, {"g", "/x"}}, keyOf); ok {
		t.Fatal("无重复键不应报告重复")
	}
	if _, ok := DuplicateKeyed([]item(nil), keyOf); ok {
		t.Fatal("空清单不应报告重复")
	}
	dup, ok := DuplicateKeyed([]item{{"h", "/x"}, {"g", "/y"}, {"g", "/y"}, {"h", "/x"}}, keyOf)
	if !ok || dup != [2]string{"g", "/y"} {
		t.Fatalf("应返回首个重复键, got %v %v", dup, ok)
	}
}
