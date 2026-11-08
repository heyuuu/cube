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
	K   string
	V   string
	Bad bool
}

// newTestStore K 去空白后为唯一键；Bad 项不合规。
func newTestStore(ms *memStore[[]item]) *KeyedSliceStore[item] {
	return NewKeyedSliceStore(KeyedSliceStoreOpts[item]{
		Name:  "测试项",
		Store: ms,
		KeyOf: func(it item) string { return it.K },
		Normalize: func(it item) (item, error) {
			it.K = strings.TrimSpace(it.K)
			if it.Bad {
				return it, errors.New("不合规")
			}
			return it, nil
		},
	})
}

func TestSaveAll(t *testing.T) {
	t.Run("提交清单即落盘结果，顺序即落盘序", func(t *testing.T) {
		ms := (&memStore[[]item]{}).set([]item{{K: "a", V: "1"}, {K: "b", V: "2"}})
		if err := newTestStore(ms).SaveAll([]item{{K: "c", V: "3"}, {K: "a", V: "9"}}); err != nil {
			t.Fatalf("save-all 失败: %v", err)
		}
		if got := ms.v; len(got) != 2 || got[0] != (item{K: "c", V: "3"}) || got[1] != (item{K: "a", V: "9"}) {
			t.Fatalf("应按提交序整体覆盖: %+v", got)
		}
	})

	t.Run("单项不合规整体拒绝不落盘", func(t *testing.T) {
		ms := (&memStore[[]item]{}).set([]item{{K: "a"}})
		if err := newTestStore(ms).SaveAll([]item{{K: "x"}, {K: "y", Bad: true}}); err == nil {
			t.Fatal("单项不合规应报错")
		}
		if ms.saves != 0 {
			t.Fatal("失败的 save-all 不应落盘")
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
	ms := (&memStore[[]item]{}).set([]item{{K: "a", V: "s1"}})
	if err := newTestStore(ms).Save(item{K: " a ", V: "s2"}); err != nil {
		t.Fatalf("save 失败: %v", err)
	}
	if len(ms.v) != 1 || ms.v[0].V != "s2" {
		t.Fatalf("规范化后同键应原位替换: %+v", ms.v)
	}
}

func TestDeleteIdempotent(t *testing.T) {
	ms := (&memStore[[]item]{}).set([]item{{K: "a"}})
	s := newTestStore(ms)
	if err := s.Delete(item{K: "a", V: "其他字段不参与匹配"}); err != nil || len(ms.v) != 0 {
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

// OnChange 在每次实际落盘后触发（入参为落盘数据集），未写入的幂等路径不触发，错误透传调用方
func TestOnChange(t *testing.T) {
	var got []item
	var calls int
	ms := (&memStore[[]item]{}).set([]item{{K: "a"}, {K: "b"}})
	s := NewKeyedSliceStore(KeyedSliceStoreOpts[item]{
		Name:  "测试项",
		Store: ms,
		KeyOf: func(it item) string { return it.K },
		OnChange: func(values []item) error {
			calls++
			got = values
			return nil
		},
	})

	if err := s.Save(item{K: "c"}); err != nil {
		t.Fatalf("save 失败: %v", err)
	}
	if calls != 1 || len(got) != 3 {
		t.Fatalf("save 落盘后应触发 OnChange: calls=%d values=%+v", calls, got)
	}

	if err := s.Delete(item{K: "zz"}); err != nil {
		t.Fatalf("delete 失败: %v", err)
	}
	if calls != 1 {
		t.Fatalf("幂等删除未写入不应触发 OnChange: calls=%d", calls)
	}

	if err := s.Delete(item{K: "b"}); err != nil {
		t.Fatalf("delete 失败: %v", err)
	}
	if calls != 2 || len(got) != 2 {
		t.Fatalf("delete 落盘后应触发 OnChange: calls=%d values=%+v", calls, got)
	}

	if err := s.SaveAll([]item{{K: "a"}}); err != nil {
		t.Fatalf("save-all 失败: %v", err)
	}
	if calls != 3 || len(got) != 1 {
		t.Fatalf("save-all 落盘后应触发 OnChange: calls=%d values=%+v", calls, got)
	}

	if err := s.Retain(func(item) bool { return true }); err != nil {
		t.Fatalf("retain 失败: %v", err)
	}
	if calls != 3 {
		t.Fatalf("retain 无删动不应触发 OnChange: calls=%d", calls)
	}
}

func TestOnChangeErrorPropagates(t *testing.T) {
	wantErr := errors.New("级联失败")
	s := NewKeyedSliceStore(KeyedSliceStoreOpts[item]{
		Name:     "测试项",
		Store:    &memStore[[]item]{},
		KeyOf:    func(it item) string { return it.K },
		OnChange: func([]item) error { return wantErr },
	})
	if err := s.Save(item{K: "a"}); !errors.Is(err, wantErr) {
		t.Fatalf("OnChange 错误应透传给调用方, got %v", err)
	}
}
