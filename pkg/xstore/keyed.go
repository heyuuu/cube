package xstore

import (
	"fmt"
	"strings"

	"github.com/heyuuu/x/store"
	"github.com/heyuuu/x/xstd/xslices"

	"cube/pkg/x"
)

// Keyed 以字符串唯一键标识自身的清单条目。
type Keyed interface {
	Key() string
}

// KeyedSliceStoreOpts 声明 KeyedSliceStore 的配置
type KeyedSliceStoreOpts[T Keyed] struct {
	// Name 实体名（中文，错误文案用，如 "scan 规则"）
	Name string
	// Store 节的存取通道（通常 store.SectionStore；测试与缓存实现可替换）
	Store store.Store[[]T]
	// LoadConvert 读取时转换；缺省为原样返回
	LoadConvert func([]T) []T
}

// KeyedSliceStore 按唯一键组织的清单存储（save / delete / save-all 语义）。
type KeyedSliceStore[T Keyed] struct {
	name        string
	store       store.Store[[]T]
	loadConvert func([]T) []T
}

// NewKeyedSliceStore 按配置构造 KeyedSliceStore，Name / Store 缺失视为编程错误直接 panic。
func NewKeyedSliceStore[T Keyed](opts KeyedSliceStoreOpts[T]) *KeyedSliceStore[T] {
	if len(strings.TrimSpace(opts.Name)) == 0 {
		panic("name is empty")
	}
	if opts.Store == nil {
		panic("store is nil")
	}
	if opts.LoadConvert == nil {
		opts.LoadConvert = identity
	}

	return &KeyedSliceStore[T]{
		name:        opts.Name,
		store:       opts.Store,
		loadConvert: opts.LoadConvert,
	}
}

// All 读全部条目（经 LoadConvert 转换）。
func (s *KeyedSliceStore[T]) All() []T {
	return s.loadConvert(s.store.LoadOrZero())
}

// Retain 按谓词保留条目，无删动时不写入。
func (s *KeyedSliceStore[T]) Retain(f func(T) bool) error {
	values := s.All()
	rest := xslices.Filter(values, f)
	if len(rest) == len(values) {
		return nil
	}
	return s.store.Save(rest)
}

// Save 命中唯一键原位替换，否则追加到末尾。
func (s *KeyedSliceStore[T]) Save(v T) error {
	values := s.All()

	values = x.UpsertKeyed(values, T.Key, v)
	return s.store.Save(values)
}

// Delete 按 v 的唯一键删除条目（只按键匹配，不比对其他字段）。
func (s *KeyedSliceStore[T]) Delete(v T) error {
	key := v.Key()
	values := s.All()
	rest, removed := x.RemoveKeyed(values, T.Key, key)
	if !removed {
		return fmt.Errorf("未找到指定 %s: %s", s.name, key)
	}
	return s.store.Save(rest)
}

// SaveAll 以 values 整体覆盖（顺序即落盘序），键重复拒绝写入。
func (s *KeyedSliceStore[T]) SaveAll(values []T) error {
	if dup, ok := x.DuplicateKeyed(values, T.Key); ok {
		return fmt.Errorf("%s 重复: %s", s.name, dup)
	}
	if values == nil {
		values = []T{}
	}
	return s.store.Save(values)
}

// Find 按唯一键查找条目。
func (s *KeyedSliceStore[T]) Find(key string) (T, bool) {
	values := s.All()
	for _, value := range values {
		if value.Key() == key {
			return value, true
		}
	}

	var zero T
	return zero, false
}

// Exists 判断唯一键是否存在。
func (s *KeyedSliceStore[T]) Exists(key string) bool {
	_, exists := s.Find(key)
	return exists
}
