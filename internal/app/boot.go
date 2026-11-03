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
	"cube/web"
)

// 注册清单：新增业务 domain 只改本文件（gen 脚本以文件内注入点标记为锚点自动追加）。

func bootFactories() []any {
	return []any{
		/* services */
		project.NewService,
		opener.NewService,
		usage.NewService,
		workbench.NewService,
		template.NewService,
		forge.NewService,

		/* <<service-factory inject point>> */
		/* 上面是 generate 脚本注入点标识，不要删除 */

		/* handlers */
		handlers.NewSystemHandler,                     // 系统端点
		di.Value(handlers.NewStaticHandler(web.FS())), // 静态资源端点
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

// ProjectService 返回 project 服务单例（Init 后可用）。
func (a *App) ProjectService() *project.Service { return mustGet[*project.Service](a) }

// OpenerService 返回 opener 服务单例（Init 后可用）。
func (a *App) OpenerService() *opener.Service { return mustGet[*opener.Service](a) }

// UsageService 返回 usage 服务单例（Init 后可用）。
func (a *App) UsageService() *usage.Service { return mustGet[*usage.Service](a) }

// TemplateService 返回 template 服务单例（Init 后可用）。
func (a *App) TemplateService() *template.Service { return mustGet[*template.Service](a) }

// ForgeService 返回 forge 服务单例（Init 后可用）。
func (a *App) ForgeService() *forge.Service { return mustGet[*forge.Service](a) }
