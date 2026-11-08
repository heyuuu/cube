package xstore

import (
	"fmt"
	"log/slog"
	"slices"
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
	Normalize func(new T) (T, error)
	// ConflictCheck 校验新元素和已有元素是否冲突
	ConflictCheck func(new T, exists T) error
	// OnChange 数据集成功落盘后回调（入参为落盘后的数据集），用于级联维护依赖它的数据；
	// 同步执行，错误透传给触发写入的调用方。未发生实际写入（幂等删除未命中、Retain 无删动）时不触发
	OnChange func(values []T) error
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
	onChange      func(values []T) error
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
		onChange:      opts.OnChange,
	}
}

// save 所有写盘的唯一出口：落盘成功后触发 OnChange，保证任何写路径都不会漏掉级联。
func (s *KeyedSliceStore[T]) save(values []T) error {
	if err := s.store.Save(values); err != nil {
		return err
	}
	if s.onChange != nil {
		return s.onChange(values)
	}
	return nil
}

// append 把 item 规范化后并入 exists（同键原位替换，否则追加到尾部），成功返回 (新数据集, nil)；
// 失败返回 (exists, 错误信息)。
func (s *KeyedSliceStore[T]) append(exists []T, item T) ([]T, error) {
	// 先规范化数据项
	if s.normalize != nil {
		var err error
		if item, err = s.normalize(item); err != nil {
			return exists, err
		}
	}

	// 键在规范化之后计算：KeyOf 未必对规范化不变（如 scan 规则 path 的 Clean）
	existsIndex := s.indexOf(exists, s.keyOf(item))

	// 校验数据集冲突
	if s.conflictCheck != nil {
		for i, exist := range exists {
			if i == existsIndex {
				continue
			}
			if err := s.conflictCheck(item, exist); err != nil {
				return exists, err
			}
		}
	}

	// 更新数据
	if existsIndex >= 0 {
		exists[existsIndex] = item
		return exists, nil
	}
	return append(exists, item), nil
}

func (s *KeyedSliceStore[T]) indexOf(list []T, key string) int {
	return slices.IndexFunc(list, func(t T) bool { return s.keyOf(t) == key })
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

// Retain 按谓词保留条目，与落盘结果一致时不写入。必须对比落盘原始数据而非 All()：
// All() 已滤掉失效条目（如所属 forge 被删的 account），对比它会判定「无删动」而不落盘，
// 失效条目残留在文件里，待条件恢复（重新添加同名 forge）时复活。
func (s *KeyedSliceStore[T]) Retain(f func(T) bool) error {
	raw := s.store.LoadOrZero()
	rest := xslices.Filter(s.All(), f)
	if len(rest) == len(raw) {
		return nil
	}
	return s.save(rest)
}

// Save 命中唯一键原位替换，否则追加到末尾。
func (s *KeyedSliceStore[T]) Save(v T) (err error) {
	values := s.All()
	values, err = s.append(values, v)
	if err != nil {
		return err
	}
	return s.save(values)
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
	return s.save(rest)
}

// SaveAll 以 values 整体覆盖（顺序即落盘序）。逐项 normalize + 冲突校验，任一失败
// 整体不落盘；重复键违反「key 在数据集中唯一」的假设，整体拒绝而非静默合并。
func (s *KeyedSliceStore[T]) SaveAll(values []T) (err error) {
	normalized := make([]T, 0, len(values))
	for _, item := range values {
		next, err := s.append(normalized, item)
		if err != nil {
			return err
		}
		// append 对同键是替换语义（长度不变）——整表覆盖下意味着提交清单里键重复
		if len(next) == len(normalized) {
			return fmt.Errorf("%s 重复: %s", s.name, s.keyOf(item))
		}
		normalized = next
	}
	return s.save(normalized)
}

// Find 按唯一键查找条目。
func (s *KeyedSliceStore[T]) Find(key string) (T, bool) {
	values := s.All()
	if i := s.indexOf(values, key); i >= 0 {
		return values[i], true
	}

	var zero T
	return zero, false
}

// Exists 判断唯一键是否存在。
func (s *KeyedSliceStore[T]) Exists(key string) bool {
	_, exists := s.Find(key)
	return exists
}
