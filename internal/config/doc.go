// Package config 是配置基础设施：加载进程配置 config.json、计算数据目录下各路径。
// 所有层共享，不 import 任何领域包。
//
// 配置目录下两份 JSON 按「生命周期」切割，不要混放：
//   - config.json（config.go）：进程级配置，启动时 Load 一次、不热更，改动重启生效；
//     按 domain 分节。文件缺失降级为默认值，读取/解析失败返回错误。
//   - settings.json：运行中可变更的用户可管理数据，多节 JSON 文档，各领域只读写自己的节。
//     节级读写 API 在外部包 github.com/heyuuu/x/store（store.LoadSection / store.SaveSection）：
//     读失败一律降级为空（读路径永不阻塞调用方）；写走锁内读-改-写 + 原子写，
//     文件为坏 JSON 时拒写，避免静默覆盖其他节。本包只经 Paths 提供 settings.json 路径。
//
// Paths（paths.go）是数据目录下子路径的唯一事实源（settings.json、log/、state/、cache/），
// 调用方不得自行硬拼——目录分层语义（根放要随机器带走的重要数据 / state 放丢了可接受但不理想的
// 运行期状态 / cache 放随时可重建的纯缓存）收敛于此。
//
// 职责边界：本包只管机制，不感知 settings.json 任何节名与节内结构，
// 节名常量与节内校验归各领域包（如 opener 包的 "openers" 节）；
// 默认配置目录按环境分流（dev / prod）的判定与派生归 cmd 层与 internal/version，
// 本包只接收绝对路径（~ 与相对路径的展开由调用方完成）。
package config
