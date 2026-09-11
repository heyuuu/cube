package slicekit

import "fmt"

// 按 key 组织的列表条目三原语（upsert / remove / reorder）：
// 键类型 K 须 comparable（字符串键与 struct 键均可）；
// 错误文案的领域语义经 entity / formatKey 由调用方注入。

// UpsertKeyed 命中 key 则原位替换，否则追加到末尾。
func UpsertKeyed[T any, K comparable](list []T, keyOf func(T) K, item T) []T {
	key := keyOf(item)
	for i, cur := range list {
		if keyOf(cur) == key {
			list[i] = item
			return list
		}
	}
	return append(list, item)
}

// RemoveKeyed 删除全部命中 key 的条目，返回剩余清单与是否有删动（false = 未找到，调用方报错）。
func RemoveKeyed[T any, K comparable](list []T, keyOf func(T) K, key K) ([]T, bool) {
	rest := make([]T, 0, len(list))
	removed := false
	for _, cur := range list {
		if keyOf(cur) == key {
			removed = true
			continue
		}
		rest = append(rest, cur)
	}
	return rest, removed
}

// ReorderKeyed 按 keys 顺序重排；未列出的条目保持原相对顺序殿后，不丢数据。
// 存储清单内键重复、keys 含未知键或重复键均返回中文错误（entity 形如 "opener" / "scan 规则"，
// formatKey 渲染键用于错误文案）。
func ReorderKeyed[T any, K comparable](list []T, keyOf func(T) K, keys []K, entity string, formatKey func(K) string) ([]T, error) {
	byKey := make(map[K]T, len(list))
	for _, item := range list {
		k := keyOf(item)
		if _, dup := byKey[k]; dup {
			return nil, fmt.Errorf("存在重复键的 %s，无法重排", entity)
		}
		byKey[k] = item
	}
	seen := make(map[K]bool, len(keys))
	for _, k := range keys {
		if _, ok := byKey[k]; !ok {
			return nil, fmt.Errorf("未找到指定 %s: %s", entity, formatKey(k))
		}
		if seen[k] {
			return nil, fmt.Errorf("重排名单存在重复 %s: %s", entity, formatKey(k))
		}
		seen[k] = true
	}

	ordered := make([]T, 0, len(list))
	for _, k := range keys {
		ordered = append(ordered, byKey[k])
	}
	for _, item := range list {
		if !seen[keyOf(item)] {
			ordered = append(ordered, item)
		}
	}
	return ordered, nil
}
