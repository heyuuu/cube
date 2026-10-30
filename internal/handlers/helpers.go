package handlers

// ListResult 列表请求通用结果
type ListResult[T any] struct {
	List []T `json:"list"`
}

func listResult[T any](list []T) ListResult[T] {
	return ListResult[T]{List: list}
}
