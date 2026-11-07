package xstore

import (
	"log/slog"
	"strings"

	"github.com/heyuuu/x/store"
	"github.com/heyuuu/x/xstd/xslices"

	"cube/pkg/x"
)

// KeyedSliceStoreOpts 声明 KeyedSliceStore 的配置
type KeyedSliceStoreOpts[T any] struct {
	// Name 数据项名（错误文案用，如 "scan 规则"）
	Name string
	// Store 数据集的存取通道
	Store store.Store[[]T]
	// KeyOf 数据项唯一键计算
	KeyOf func(T) string
	// Normalize 格式化并校验数据项
	Normalize func(T) (T, error)
	// ConflictCheck 校验新元素和已有元素是否冲突
	ConflictCheck func(new T, exists T) error
}

// KeyedSliceStore 可被索引的数据结合
// 基于以下假设
// - 每个数据项都有 key，且在数据集中唯一
// - 一个数据项加入数据集的方式是优先覆盖同 key 数据项，无相同 key 数据项时添加到数据集尾部
// - 一个数据项，只要自身合规(通过 normalize 检测)，且不与其他不同 key 的数据项冲突(通过 conflictCheck 检测)，即可加入数据集
// - 一个合规数据集，删除任意若干项后还是一个合规数据集
type KeyedSliceStore[T any] struct {
	name          string
	store         store.Store[[]T]
	keyOf         func(T) string
	normalize     func(T) (T, error)
	conflictCheck func(new T, exists T) error
}

func NewKeyedSliceStore[T any](opts KeyedSliceStoreOpts[T]) *KeyedSliceStore[T] {
	if len(strings.TrimSpace(opts.Name)) == 0 {
		panic("name is empty")
	}
	if opts.Store == nil {
		panic("store is nil")
	}
	if opts.KeyOf == nil {
		panic("keyOf is nil")
	}
	return &KeyedSliceStore[T]{
		name:          opts.Name,
		store:         opts.Store,
		keyOf:         opts.KeyOf,
		normalize:     opts.Normalize,
		conflictCheck: opts.ConflictCheck,
	}
}

// append 向已有数据集追加数据，成功返回 (新数据集, nil); 失败返回 (旧数据集, 错误信息)
func (s *KeyedSliceStore[T]) append(exists []T, item T) (result []T, err error) {
	// 先规范化数据项
	if s.normalize != nil {
		item, err = s.normalize(item)
		if err != nil {
			return exists, err
		}
	}

	// 校验数据集冲突
	if s.conflictCheck != nil {
		key := s.keyOf(item)
		for _, exist := range exists {
			if s.keyOf(exist) == key {
				continue
			}
			if err := s.conflictCheck(item, exist); err != nil {
				return exists, err
			}
		}
	}

	// 更新数据
	return x.UpsertKeyed(exists, s.keyOf, item), nil
}

// All 读全部条目
func (s *KeyedSliceStore[T]) All() []T {
	raw := s.store.LoadOrZero()
	if len(raw) == 0 {
		return nil
	}

	result := make([]T, 0, len(raw))
	for i, item := range raw {
		var err error
		result, err = s.append(result, item)

		// 加载配置时，忽略错误项，仅记录日志
		if err != nil {
			slog.Warn("数据项加载错误", "name", s.name, "index", i, "err", err)
		}
	}
	return result
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
func (s *KeyedSliceStore[T]) Save(v T) (err error) {
	values := s.All()
	values, err = s.append(values, v)
	if err != nil {
		return err
	}
	return s.store.Save(values)
}

// Delete 按 v 的唯一键删除条目（只按键匹配，不比对其他字段）。
// 未命中不报错，保持幂等
func (s *KeyedSliceStore[T]) Delete(v T) error {
	key := s.keyOf(v)
	values := s.All()
	rest, removed := x.RemoveKeyed(values, s.keyOf, key)
	if !removed {
		return nil
	}
	return s.store.Save(rest)
}

// SaveAll 以 values 整体覆盖（顺序即落盘序），键重复拒绝写入。
func (s *KeyedSliceStore[T]) SaveAll(values []T) (err error) {
	normalized := make([]T, 0, len(values))
	for _, item := range values {
		normalized, err = s.append(normalized, item)
		if err != nil {
			return err
		}
	}
	return s.store.Save(values)
}

// Find 按唯一键查找条目。
func (s *KeyedSliceStore[T]) Find(key string) (T, bool) {
	values := s.All()
	for _, v := range values {
		if s.keyOf(v) == key {
			return v, true
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
