package handlers

type WithBody[T any] struct {
	Body T
}

// ListResult 列表请求通用结果
type ListResult[T any] struct {
	List []T `json:"list"`
}

func listResult[T any](list []T) *ListResult[T] {
	return &ListResult[T]{List: list}
}

// OpResult 操作无需具体结构的通用结构
type OpResult struct {
	Ok bool `json:"ok"`
}

var opSuccess = &OpResult{Ok: true}
