package xstore

// 返回自身的函数，用于函数参数
func identity[T any](v T) T {
	return v
}
