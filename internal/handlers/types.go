package handlers

// WithBody POST body 入参的通用包装：huma 约定请求体字段须挂在名为 Body 的
// 子结构上，handler 签名统一用 WithBody[T]（T 为内联 body struct 或 DTO 单项），
// 不再逐端点声明具名 Input 类型。
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

// OpResult 操作型端点（save / delete / open 等）的通用出参；需要携带数据的
// 端点另声明具名 Result 类型。
type OpResult struct {
	Ok bool `json:"ok"`
}

// opSuccess 共享的成功哨兵——成功响应内容恒等，只读不改，避免每次请求分配
var opSuccess = &OpResult{Ok: true}
