package web

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// nilCollectionsJSONFormat 替换 huma 默认 JSON 格式：把响应中的 nil 切片序列化为 []，
// nil map 序列化为 {}（标准库 v1 默认输出 null）。
//
// 这是纯展示层关注点——前端 TS 类型声明为 string[] / Record，拿到 null 会触发
// 运行时崩溃。内部业务代码无需关心 nil/empty 区别（Go 里 nil slice 本就可安全
// range/append），这层只负责给前端一个稳定的 JSON 形态。
//
// encoding/json/v2 已在 Go 1.27 转正，其默认集合语义（nil 切片/map → []/{}、
// nil 指针 → null）与本格式一致；待 huma 原生支持 v2 前，本文件继续以 v1 +
// reflect 替换模拟该语义（huma.DefaultConfig 走 v1 会输出 null，不能直接还原）。
var nilCollectionsJSONFormat = huma.Format{
	Marshal: func(w io.Writer, v any) error {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false) // 与 huma DefaultJSONFormat 行为对齐
		return enc.Encode(replaceNilCollections(v))
	},
	Unmarshal: json.Unmarshal, // 入站不改：前端发 null 仍正常解析为 nil
}

// replaceNilCollections 递归把 v 中所有 nil 切片/map 替换为对应类型的空集合。
// 返回新构造的值，不修改原对象（值类型天然隔离；struct 经由深拷贝副本）。
func replaceNilCollections(v any) any {
	if v == nil {
		return nil
	}
	return replaceNilCollectionsValue(reflect.ValueOf(v)).Interface()
}

// replaceNilCollectionsValue 是 reflect 层的递归实现，返回替换后的 reflect.Value。
func replaceNilCollectionsValue(rv reflect.Value) reflect.Value {
	// 解包指针/接口，拿到底层值；nil 指针/接口原样返回（输出 null 是合理的）。
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return rv
		}
		// 指针/接口本身保留包装，递归处理其元素后重新包装回去。
		elem := replaceNilCollectionsValue(rv.Elem())
		// 指针指向的若是新构造的副本，需更新指针；
		// 但直接 SetElem 受 CanAddr 限制——改为返回新指针更稳妥。
		if rv.Kind() == reflect.Pointer {
			out := reflect.New(elem.Type())
			out.Elem().Set(elem)
			return out
		}
		return elem
	}

	switch rv.Kind() {
	case reflect.Slice:
		if rv.IsNil() {
			return reflect.MakeSlice(rv.Type(), 0, 0) // nil → 空切片
		}
		// 非 nil 切片：逐元素递归（元素可能是 struct，内含 nil 切片字段）。
		out := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Cap())
		for i := 0; i < rv.Len(); i++ {
			out.Index(i).Set(replaceNilCollectionsValue(rv.Index(i)))
		}
		return out

	case reflect.Map:
		if rv.IsNil() {
			return reflect.MakeMap(rv.Type()) // nil → 空 map
		}
		out := reflect.MakeMapWithSize(rv.Type(), rv.Len())
		for iter := rv.MapRange(); iter.Next(); {
			out.SetMapIndex(iter.Key(), replaceNilCollectionsValue(iter.Value()))
		}
		return out

	case reflect.Struct:
		// 先整体浅拷贝再替换 exported 字段：unexported 字段（如 time.Time 的
		// wall/ext/loc）拿不到 reflect 写权限，逐字段重建会把它们清成零值——
		// 曾导致所有响应里的 time.Time 变成 0001-01-01。浅拷贝保住它们。
		out := reflect.New(rv.Type()).Elem()
		out.Set(rv)
		for i := 0; i < rv.NumField(); i++ {
			if rv.Type().Field(i).IsExported() {
				out.Field(i).Set(replaceNilCollectionsValue(rv.Field(i)))
			}
		}
		return out
	}

	return rv
}
