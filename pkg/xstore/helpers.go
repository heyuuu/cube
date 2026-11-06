// Package xstore 基于 github.com/heyuuu/x/store 的业务无关存储组合件。
package xstore

// 返回自身的函数，用于函数参数
func identity[T any](v T) T {
	return v
}
