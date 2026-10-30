// Package slicekit 切片运算小工具（Map / Filter 等纯函数，不含业务语义）。
package slicekit

// Map 逐元素映射为新切片。
func Map[S ~[]E1, E1 any, E2 any](s S, mapper func(E1) E2) []E2 {
	if len(s) == 0 {
		return nil
	}

	result := make([]E2, len(s))
	for i, item := range s {
		result[i] = mapper(item)
	}
	return result
}

// Filter 按谓词过滤（保序）。
func Filter[S ~[]E, E any](s S, filter func(E) bool) []E {
	if len(s) == 0 {
		return nil
	}

	result := make([]E, 0, len(s))
	for _, item := range s {
		if filter(item) {
			result = append(result, item)
		}
	}
	return result
}
