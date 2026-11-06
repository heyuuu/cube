package x

// 按 key 组织的列表条目原语（upsert / remove / 查重）：
// 键类型 K 须 comparable（字符串键与 struct 键均可）。

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

// DuplicateKeyed 返回清单内首个重复出现的键（ok=false 表示无重复），整表覆盖写入前查重用。
func DuplicateKeyed[T any, K comparable](list []T, keyOf func(T) K) (dup K, ok bool) {
	seen := make(map[K]bool, len(list))
	for _, item := range list {
		k := keyOf(item)
		if seen[k] {
			return k, true
		}
		seen[k] = true
	}
	return dup, false
}
