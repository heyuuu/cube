// api 子命令：在已有 domain 的 handler 文件里追加一个 api 端点。

package generate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"

	"github.com/heyuuu/x/xstd/xslices"
	"github.com/heyuuu/x/xstd/xstrings"
)

var (
	urlSegmentPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z][a-z0-9]*)*$`)
)

func checkRoute(raw string) (domain string, basename string, err error) {
	segments := xslices.Filter(strings.Split(raw, "/"), func(s string) bool { return s != "" })
	if len(segments) < 3 || segments[0] != "api" {
		return "", "", fmt.Errorf("不支持的 api route，预期 api route 为 /api/{domain}/{basename} 形式: %s", raw)
	}
	for _, segment := range segments[1:] {
		if !urlSegmentPattern.MatchString(segment) {
			return "", "", fmt.Errorf("不支持的 api route，预期 uri segment 为小写字母、数字、中划线组成，且中划线不可在首尾: %s", raw)
		}
	}

	domain = segments[1]
	basename = strings.Join(segments[2:], "/")
	return domain, basename, nil
}

// GenAPI 追加端点 (GET|POST) /api/{domain}/{base-name} 到 {domain}_handler.go：
// Register 方法末尾插入注册行，文件尾部追加 Input/Output 类型与 handler 方法。
func GenAPI(method, route string) error {
	if err := checkMethod(method); err != nil {
		return err
	}

	domain, basename, err := checkRoute(route)
	if err != nil {
		return err
	}

	// 预处理所需参数
	method = strings.ToLower(method)
	domain = strings.ToLower(domain)
	basename = strings.ToLower(basename)
	route = "/api/" + domain + "/" + basename                        // 完整路由名
	routeMethod := xstrings.UpperFirst(method)                       // route 方法名
	handlerMethod := xstrings.CamelCase(basename)                    // handler 方法名
	resultType := xstrings.CamelCase(domain+"/"+basename) + "Result" // Result 类型名

	// 处理模板文件
	handlerFile := "internal/handlers/" + domain + "_handler.go"
	src, err := os.ReadFile(handlerFile)
	if err != nil {
		return fmt.Errorf("读取 %s 失败（domain 尚未生成？先用 domain 子命令生成）: %w", handlerFile, err)
	}
	text := string(src)

	// 查重：路由或任一同名标识符已存在即拒绝，防重复生成
	for _, ident := range []string{
		route,
		resultType,
		handlerMethod + "(",
	} {
		if strings.Contains(text, ident) {
			return fmt.Errorf("%s 已包含 %s，拒绝重复生成", handlerFile, strings.Trim(ident, `"(`))
		}
	}

	recv, handlerType, insertAt, err := findRegister(text)
	if err != nil {
		return fmt.Errorf("%s: %w", handlerFile, err)
	}
	data := map[string]string{
		"method":        method,
		"domain":        domain,
		"basename":      basename,
		"upperMethod":   strings.ToUpper(method),
		"route":         route,
		"routeMethod":   routeMethod,
		"handlerMethod": handlerMethod,
		"resultType":    resultType,
		"recv":          recv,
		"handlerType":   handlerType,
	}

	// 先插注册行（偏移基于原文，插入不影响其前内容），再在文件尾追加端点代码
	registerLine, err := renderTplFile("api_register.gotmpl", data)
	if err != nil {
		return err
	}
	text = text[:insertAt] + "\t" + registerLine + "\n" + text[insertAt:]

	appendCode, err := renderTplFile("api_append.gotmpl", data)
	if err != nil {
		return err
	}
	text += "\n" + appendCode + "\n"

	if err := writeFormatted(handlerFile, text); err != nil {
		return err
	}
	fmt.Printf("已生成 %s %s → %s\n", strings.ToUpper(method), route, handlerFile)
	return nil
}

// findRegister 在 handler 源码里定位 Register 方法，返回接收者变量名、接收者类型名
// 与方法体收尾 `}` 的字节偏移——在其前插入等价于在方法末尾追加一条注册语句。
// 接收者变量名与类型名取自源码而非拼约定，handler 被重命名后生成代码依然可编译。
func findRegister(src string) (recv, typeName string, insertAt int, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "handler.go", src, 0)
	if err != nil {
		return "", "", 0, fmt.Errorf("解析失败（文件有语法错误？）: %w", err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Register" || fn.Recv == nil || len(fn.Recv.List) == 0 || fn.Body == nil {
			continue
		}
		typeExpr := fn.Recv.List[0].Type
		if star, ok := typeExpr.(*ast.StarExpr); ok {
			typeExpr = star.X // 指针接收者取 * 后的类型名
		}
		id, ok := typeExpr.(*ast.Ident)
		if !ok {
			continue
		}
		if len(fn.Recv.List[0].Names) > 0 {
			recv = fn.Recv.List[0].Names[0].Name
		} else {
			recv = "h" // 接收者未命名时兜底
		}
		return recv, id.Name, fset.Position(fn.Body.Rbrace).Offset, nil
	}
	return "", "", 0, fmt.Errorf("找不到带接收者的 Register 方法")
}
