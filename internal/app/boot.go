package app

import (
	"github.com/heyuuu/x/di"

	"cube/internal/domain/forge"
	"cube/internal/domain/opener"
	"cube/internal/domain/project"
	"cube/internal/domain/template"
	"cube/internal/domain/usage"
	"cube/internal/domain/workbench"
	"cube/internal/handlers"
	"cube/internal/server"
	"cube/web"
)

// 注册清单：新增业务 domain 只改本文件（gen 脚本以文件内注入点标记为锚点自动追加）。

func serviceFactories() []any {
	return []any{
		project.NewServiceInject,
		opener.NewServiceInject,
		usage.NewServiceInject,
		workbench.NewServiceInject,
		template.NewServiceInject,
		forge.NewServiceInject,

		/* <<service-factory inject point>> */
		/* 上面是 generate 脚本注入点标识，不要删除 */
	}
}

func handlerFactories() []any {
	return []any{
		/* 系统端点 */
		server.NewSystemHandler,
		/* 静态资源端点 */
		di.Value(server.NewStaticHandler(web.FS())),

		/* 业务 handler */
		handlers.NewConfigHandler,
		handlers.NewProjectHandler,
		handlers.NewOpenerHandler,
		handlers.NewMdHandler,
		handlers.NewIconHandler,
		handlers.NewWorkbenchHandler,
		handlers.NewForgeHandler,
		handlers.NewTemplateHandler,
		handlers.NewUsageHandler,

		/* <<handler-factory inject point>> */
		/* 上面是 generate 脚本注入点标识，不要删除 */
	}
}
