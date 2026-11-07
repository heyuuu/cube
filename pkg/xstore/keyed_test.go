package xstore

import (
	"errors"
	"strings"
	"testing"
)

// memStore 内存版 store.Store，记录落盘次数供断言「是否写入」。
type memStore[T any] struct {
	v     T
	saves int
}

func (m *memStore[T]) Save(v T) error       { m.v = v; m.saves++; return nil }
func (m *memStore[T]) Load() (T, error)     { return m.v, nil }
func (m *memStore[T]) LoadOrZero() T        { return m.v }
func (m *memStore[T]) set(v T) *memStore[T] { m.v = v; return m }

type item struct {
	K      string
	Secret string
	Bad    bool
}

const masked = "***"

// newTestStore K 去空白后为唯一键；Bad 项不合规；Secret 提交掩码值时沿用同键旧值（同 forge account token）。
func newTestStore(ms *memStore[[]item]) *KeyedSliceStore[item] {
	return NewKeyedSliceStore(KeyedSliceStoreOpts[item]{
		Name:  "测试项",
		Store: ms,
		KeyOf: func(it item) string { return it.K },
		NormalizeWithExists: func(it item, old *item) (item, error) {
			it.K = strings.TrimSpace(it.K)
			if it.Bad {
				return it, errors.New("不合规")
			}
			if it.Secret == masked {
				if old == nil {
					return it, errors.New("不存在，不能提交掩码")
				}
				it.Secret = old.Secret
			}
			return it, nil
		},
	})
}

func TestSaveAll(t *testing.T) {
	t.Run("同键旧值取自落盘数据（掩码沿用）且顺序即落盘序", func(t *testing.T) {
		ms := (&memStore[[]item]{}).set([]item{{K: "a", Secret: "s1"}, {K: "b", Secret: "s2"}})
		if err := newTestStore(ms).SaveAll([]item{{K: "b", Secret: masked}, {K: "a", Secret: masked}}); err != nil {
			t.Fatalf("save-all 失败: %v", err)
		}
		if got := ms.v; len(got) != 2 || got[0] != (item{K: "b", Secret: "s2"}) || got[1] != (item{K: "a", Secret: "s1"}) {
			t.Fatalf("应按提交序落盘且掩码沿用旧值: %+v", got)
		}
	})

	t.Run("新条目提交掩码报错", func(t *testing.T) {
		ms := &memStore[[]item]{}
		if err := newTestStore(ms).SaveAll([]item{{K: "new", Secret: masked}}); err == nil {
			t.Fatal("新条目提交掩码应报错")
		}
	})

	t.Run("规范化后同键重复整体拒绝不落盘", func(t *testing.T) {
		ms := (&memStore[[]item]{}).set([]item{{K: "a"}})
		err := newTestStore(ms).SaveAll([]item{{K: "x"}, {K: " x "}})
		if err == nil || !strings.Contains(err.Error(), "重复") {
			t.Fatalf("重复键应报错, got %v", err)
		}
		if ms.saves != 0 {
			t.Fatal("失败的 save-all 不应落盘")
		}
	})

	t.Run("落盘规范化结果", func(t *testing.T) {
		ms := &memStore[[]item]{}
		if err := newTestStore(ms).SaveAll([]item{{K: " a "}}); err != nil {
			t.Fatalf("save-all 失败: %v", err)
		}
		if ms.v[0].K != "a" {
			t.Fatalf("应落盘规范化后的值: %+v", ms.v)
		}
	})
}

// 键在规范化之后计算：未规范化的同键提交应替换而非追加
func TestSaveKeyAfterNormalize(t *testing.T) {
	ms := (&memStore[[]item]{}).set([]item{{K: "a", Secret: "s1"}})
	if err := newTestStore(ms).Save(item{K: " a ", Secret: "s2"}); err != nil {
		t.Fatalf("save 失败: %v", err)
	}
	if len(ms.v) != 1 || ms.v[0].Secret != "s2" {
		t.Fatalf("规范化后同键应原位替换: %+v", ms.v)
	}
}

func TestDeleteIdempotent(t *testing.T) {
	ms := (&memStore[[]item]{}).set([]item{{K: "a"}})
	s := newTestStore(ms)
	if err := s.Delete(item{K: "a", Secret: "其他字段不参与匹配"}); err != nil || len(ms.v) != 0 {
		t.Fatalf("应按键删除: err=%v %+v", err, ms.v)
	}
	saves := ms.saves
	if err := s.Delete(item{K: "a"}); err != nil || ms.saves != saves {
		t.Fatalf("未命中应幂等成功且不写入: err=%v saves=%d", err, ms.saves)
	}
}

// Retain 对比落盘原始数据：All() 已滤掉的失效条目也要借此落盘清除
func TestRetainPersistsFilteredInvalid(t *testing.T) {
	ms := (&memStore[[]item]{}).set([]item{{K: "a"}, {K: "b", Bad: true}})
	s := newTestStore(ms)
	if err := s.Retain(func(item) bool { return true }); err != nil {
		t.Fatalf("retain 失败: %v", err)
	}
	if len(ms.v) != 1 || ms.v[0].K != "a" {
		t.Fatalf("失效条目应被落盘清除: %+v", ms.v)
	}
	saves := ms.saves
	if err := s.Retain(func(item) bool { return true }); err != nil || ms.saves != saves {
		t.Fatalf("与落盘一致时不应写入: err=%v saves=%d", err, ms.saves)
	}
}
