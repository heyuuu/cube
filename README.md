# cube

## 介绍

开发者使用的本地多项目管理工具。

- [1.0 版本]() — 采用 php 开发的纯命令行版本
- 2.* 版本 — 使用 go 语言重写的升级版本
- 3.* 版本 — 架构与工程化的全面重构

## v3 主要变更

- **架构调整**：目录从分层结构改为按领域划分；展平包，移除 `internal` 包嵌套
- **领域模型简化**：合并 project 域的实体，`workspace` 降为 `ScanRule` 值对象，`remote` 降为 `CloneRule` 值对象；合并 project 领域功能，统一以 `ProjectService` 对外暴露
- **依赖精简**：移除 `wire` 依赖注入，简化 app 包
- **命名调整**：module path 改为 `github.com/heyuuu/cube`；`application` 更名为 `opener`
- **Web 重构**：使用 `huma` 包重构 server，仅保留 http 支持，新增 `openapi.json` 文档输出；`server` 命令支持 `port` 参数，新增 `server openapi` 子命令
- **模糊匹配升级**：`matcher` 更名为 `fuzzy` 包并重构，改用 DP 算法提升匹配效率，增加特殊 bonus 分值优化长路径搜索排序
- **配置调整**：默认目录改为 `~/.cube/`
- **Go 版本**：提升至 1.25.0
