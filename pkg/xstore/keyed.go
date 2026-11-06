package xstore

import (
	"fmt"
	"strings"

	"github.com/heyuuu/x/store"
	"github.com/heyuuu/x/xstd/xslices"

	"cube/pkg/x"
)

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

type KeyedSliceStore[T Keyed] struct {
	name        string
	store       store.Store[[]T]
	loadConvert func([]T) []T
}

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

func (s *KeyedSliceStore[T]) All() []T {
	return s.loadConvert(s.store.LoadOrZero())
}

func (s *KeyedSliceStore[T]) Retain(f func(T) bool) error {
	values := s.All()
	rest := xslices.Filter(values, f)
	if len(rest) == len(values) {
		return nil
	}
	return s.store.Save(rest)
}

func (s *KeyedSliceStore[T]) Save(v T) error {
	values := s.All()

	values = x.UpsertKeyed(values, T.Key, v)
	return s.store.Save(values)
}

func (s *KeyedSliceStore[T]) Delete(key string) error {
	values := s.All()
	rest, removed := x.RemoveKeyed(values, T.Key, key)
	if !removed {
		return fmt.Errorf("未找到指定 %s: %s", s.name, key)
	}
	return s.store.Save(rest)
}

func (s *KeyedSliceStore[T]) Reorder(keys []string) error {
	values := s.All()
	ordered, err := x.ReorderKeyed(values, T.Key, keys, s.name, identity)
	if err != nil {
		return err
	}
	return s.store.Save(ordered)
}

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

func (s *KeyedSliceStore[T]) Exists(key string) bool {
	_, exists := s.Find(key)
	return exists
}
