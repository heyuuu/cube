---
name: rfc-continue
description: 推进 xdiff 项目中一个已有 RFC：完善需求进入讨论、沉淀讨论结论并定稿（accepted）、进入或继续开发（developing）、挂起与恢复。用户说"继续 rfc-005"、"讨论 xxx"、"005 方案定了"、"开始开发 002"、"挂起/恢复 rfc-003"，或想以任何方式推进某个需求时使用。
---

# RFC 推进

把一个已有 RFC 往前推。机制全文见 `docs/rfcs/README.md`；两者冲突时以 README 为准。

## 定位 RFC

- 用户给了编号或名字 → 匹配 `docs/rfcs/rfc-{编号}-*` 目录。
- 没给 → 读各 RFC.md 的 front-matter，列出 `id | title | status` 供用户选择，不要猜。

定位后先读该 RFC.md 全文再动手。

## 状态机（只允许表内流转）

```
draft → discussion → accepted → developing → done
                    ├──→ parked ──→ (回到 discussion)
                    └────────────→ rejected（任意阶段可）
```

| 当前状态 | 推进动作 |
|----------|----------|
| `draft` | 打磨需求描述与验收标准，状态改 `discussion` |
| `discussion` | 讨论结论沉淀进「方案讨论」；方案定稿后填写「最终方案」，状态改 `accepted` |
| `accepted` | 状态改 `developing`，然后按方案开始编码 |
| `developing` | 继续实现；实现与文档的偏差回写「实现记录」，方案本身变了就同步修订「最终方案」 |
| `parked` | 确认挂起原因解除后，改回 `discussion` 继续 |
| `done` / `rejected` | 终态：告知用户没有下一步；衍生需求走 rfc-create 新建 |

## 关键红线

- **状态未到 `accepted` 不得写实现代码**——这是项目强制流程，用户催也不行，先定稿方案。
- 每次修改 RFC.md，front-matter 的 `updated` 同步为今天。
- 转 `parked` / `rejected` 必须在正文记录原因（为什么挂起 / 为什么放弃），否则状态改了信息就丢了。
- 开发完成（验收标准全部真实达成）后引导走 rfc-complete 收尾，不要自己直接改 `done`——收尾有核对步骤。

## 各阶段产出要求

**discussion 沉淀**：「方案讨论」写候选方案与取舍，被否决的方案保留并记否决原因——这些信息丢了就没法回顾为什么选现在这条路。

**accepted 定稿**：「最终方案」写到开发可照做的粒度：接口形态、类型/协议定义、目录位置、关键决策及其理由。只写一句结论不算定稿。

**developing 编码**：遵循仓库 `AGENTS.md`（后端分层、中文注释、路径绝对化等）；动前端前先读 `web/AGENTS.md`。实现过程中发现方案不合理，先改文档再继续写——以实际为准回写文档。
