package opener

import (
	"fmt"
	"slices"

	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/config"
	"cube/internal/server"
	"cube/pkg/fuzzy"
)

// Service opener 领域服务：openers / openerIntents 两节现读（直读不缓存，改完即生效）+ 模糊检索。
type Service struct {
	openersStore     OpenersStore
	intentInfosStore IntentInfosStore
	executor         Executor // Open 启动子进程的执行器（环境归属 Service；测试经 NewServiceWithExecutor 注入 fake）
	baseURL          string   // 站内路由基地址（如 http://127.0.0.1:6001），装配注入
}

// NewService di 装配入口：从 config 组装 settings 路径与 baseURL，用默认执行器。
func NewService(cfg *config.Config) *Service {
	return NewServiceWithExecutor(cfg, nil)
}

// NewServiceWithExecutor 与 NewService 同构，但注入自定义 Executor
// （跨包测试的 fake 注入口；executor 传 nil 则用默认执行器）。
func NewServiceWithExecutor(cfg *config.Config, executor Executor) *Service {
	if executor == nil {
		executor = NewDefaultExecutor()
	}
	openersStore := newOpenersStore(cfg.SettingsFile())
	intentInfosStore := newIntentInfosStore(cfg.SettingsFile())

	// openers 节每次落盘后级联清理 intents 失效引用（Save/SaveAll/Delete 共用此通道）
	openersStore.OnChange(func(openers []Opener) error {
		intentInfos, changed := normalizeIntentInfos(intentInfosStore.All(), openers)
		if changed {
			return intentInfosStore.SaveAll(intentInfos)
		}
		return nil
	})

	return &Service{
		openersStore:     openersStore,
		intentInfosStore: intentInfosStore,
		executor:         executor,
		baseURL:          server.BaseURL(cfg.Server),
	}
}

// openers 现读 settings.json 的 openers 节（store 逐条经 validateOpener 归一化，title 补默认）。
// store.LoadSection 已把文件级/节级坏数据降级为空；条目级坏数据跳过不阻断
// （配置错误不应让 list 等命令不可用），与旧 config 时代行为一致。
func (s *Service) openers() []Opener {
	return s.openersStore.All()
}

// AllOpeners 返回全部 opener（坏条目已跳过，不阻断）。
func (s *Service) AllOpeners() []Opener { return s.openers() }

// RoleOpeners 返回声明了指定 role 的 opener。
func (s *Service) RoleOpeners(role Role) []Opener {
	return xslices.Filter(s.openers(), func(o Opener) bool {
		return slices.Contains(o.Roles(), role)
	})
}

// SearchAll 按名称模糊检索全部 opener。
func (s *Service) SearchAll(query string) []Opener {
	return fuzzy.MatchBy(query, s.openers(), func(o Opener) string { return o.Name }, nil)
}

// SearchFor 在指定 role 的 opener 中按名称模糊检索。
func (s *Service) SearchFor(role Role, query string) []Opener {
	return fuzzy.MatchBy(query, s.RoleOpeners(role), func(o Opener) string { return o.Name }, nil)
}

// FindByName 按名称精确查找 opener，未命中返回 nil。
func (s *Service) FindByName(name string) *Opener {
	for _, o := range s.openers() {
		if o.Name == name {
			return &o
		}
	}
	return nil
}

// Open 用指定 opener 以指定 role 打开目标：role 支持校验后经 runAction 组装命令
// （槽位校验在内）并启动子进程。执行环境只出现在这里——Opener 是纯数据值对象，
// 不带任何进程设施。
func (s *Service) Open(o Opener, role Role, slotArgs ...string) error {
	action, ok := o.Actions[role]
	if !ok {
		return fmt.Errorf("opener %s 不支持 %s", o.Name, role)
	}

	return runAction(s.executor, role, action, slotArgs, s.baseURL)
}

// SaveOpener 新增或按名替换一条 opener。
// 写前先走 validateOpener 校验与归一化——坏数据返回中文错误、落不了文件
// （提案「校验收敛在读写边界」的写侧）。
func (s *Service) SaveOpener(spec Opener) error {
	return s.openersStore.Save(spec)
}

// SaveAllOpeners 以提交清单整节覆盖 openers 节。列表顺序即展示顺序（CLI/项目页快捷入口
// 均按此序），前端拖拽排序落库走这里。逐项构造校验 + name 查重，任一失败整体不落盘；
// 读侧不可见的坏条目随覆盖丢弃。被移除的 opener 连带清理 openerIntents 引用（同 DeleteOpener）。
func (s *Service) SaveAllOpeners(specs []Opener) error {
	return s.openersStore.SaveAll(specs)
}

// DeleteOpener 按唯一键（name）删除一条 opener；未命中幂等成功。
func (s *Service) DeleteOpener(spec Opener) error {
	return s.openersStore.Delete(spec)
}

// intentInfos 现读全部意图条目（稠密固定序；不含 Intents() 的缺省候选合成）。
func (s *Service) intentInfos() []IntentInfo {
	return s.intentInfosStore.All()
}

// intentInfo 取单个意图条目（先校验 intent 合法）。稠密存储下合法 intent 恒存在，
// 「未配置」分支实际不可达，仅作 Find 契约的防御。
func (s *Service) intentInfo(intent Intent) (IntentInfo, error) {
	var zero IntentInfo
	if _, err := ParseIntent(string(intent)); err != nil {
		return zero, err
	}

	info, ok := s.intentInfosStore.Find(string(intent))
	if !ok {
		return zero, fmt.Errorf("未配置 intent %s", intent)
	}

	return info, nil
}

// Intents 返回全部意图状态（固定序）。唯一的读侧合成是缺省候选：openers 为空时
// 回退为「声明了对应 role 的全部 opener」（openers 节顺序）；失效引用不在读侧
// 过滤——清理收敛在 openers 落盘的 OnChange 级联，失效默认由 DefaultOpener 调用时校验。
func (s *Service) Intents() []IntentInfo {
	infos := s.intentInfosStore.All()

	for i, info := range infos {
		if len(info.Openers) == 0 {
			role := info.Intent.Role()
			infos[i].Openers = xslices.Map(s.RoleOpeners(role), openerName)
		}
	}

	return infos
}

// DefaultOpener 返回该 intent 的默认 opener；未配置或失效时返回中文错误
// （CLI 不带 -o 时直接走这里，报错即引导用户去配置）。
func (s *Service) DefaultOpener(intent Intent) (*Opener, error) {
	info, err := s.intentInfo(intent)
	if err != nil {
		return nil, err
	}

	if info.DefaultOpener == "" {
		return nil, fmt.Errorf("intent %s 未配置默认 opener", intent)
	}

	o := s.FindByName(info.DefaultOpener)
	if o == nil {
		return nil, fmt.Errorf("intent %s 的默认 opener 不存在: %s", intent, info.DefaultOpener)
	}
	if !o.SupportsRole(intent.Role()) {
		return nil, fmt.Errorf("intent %s 的默认 opener 并不支持此 intent: %s", intent, info.DefaultOpener)
	}
	return o, nil
}

// SaveIntentDefault 设置某 intent 的默认 opener。写侧校验：intent 合法 + opener
// 存在且声明了对应 role（必死配置不落盘）；其余条目原样保留。
func (s *Service) SaveIntentDefault(intent Intent, openerName string) error {
	info, err := s.intentInfo(intent)
	if err != nil {
		return err
	}

	o := s.FindByName(openerName)
	if o == nil {
		return fmt.Errorf("opener %s 不存在，无法作为 %s 的默认", openerName, intent)
	}
	if !o.SupportsRole(intent.Role()) {
		return fmt.Errorf("opener %s 并不支持 intent %s 的 role(%s)", openerName, intent, intent.Role())
	}

	info.DefaultOpener = openerName
	return s.intentInfosStore.Save(info)
}

// DeleteIntentDefault 清除某 intent 的默认 opener（候选 openers 清单如有则保留）；
// 幂等——未配置默认时也不报错。
func (s *Service) DeleteIntentDefault(intent Intent) error {
	info, err := s.intentInfo(intent)
	if err != nil {
		return err
	}

	if info.DefaultOpener == "" {
		return nil
	}

	info.DefaultOpener = ""
	return s.intentInfosStore.Save(info)
}
