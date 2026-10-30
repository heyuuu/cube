---
name: rfc-validate
description: 校验 xdiff 项目 docs/rfcs/ 下 RFC 文档的一致性与规范性：目录命名与编号、front-matter 字段、状态枚举、日期、必需章节、占位符是否已填。用户说"校验 rfc"、"检查 rfcs"、"rfc 有没有问题"，或在 RFC 提交前想自查时使用。
---

# RFC 校验

校验 `docs/rfcs/` 下 RFC 文档的一致性。分两层：先跑脚本做机械检查，再由你补语义检查。

## 1. 运行脚本（在仓库根目录）

```bash
python3 .agents/skills/rfc-validate/scripts/check.py docs/rfcs
```

只查单个 RFC 时直接传目录：

```bash
python3 .agents/skills/rfc-validate/scripts/check.py docs/rfcs/rfc-005-目录对比页
```

脚本覆盖：目录命名与编号（唯一、跳号警告）、RFC.md 存在性、front-matter 字段齐全、id 与目录编号一致、status 合法枚举、日期格式及先后、六个必需章节、H1 与 title 一致、accepted 后「最终方案」仍占位、done 的「实现记录」/遗留问题/验收标准勾选。发现错误退出码为 1。

## 2. 语义检查（脚本查不了的，逐个过）

- `parked` / `rejected`：正文是否记录了原因。
- `done`：「实现记录」「遗留问题」是否言之有物，而不是模板引导语的复述。
- `updated` 与内容是否明显脱节（正文有大量新内容但 updated 很旧）→ 提醒同步。

## 3. 处理结果

- **机械性错误直接修复**（updated 未同步、章节标题笔误、id 不一致、status 拼错），修完复跑脚本确认清零。
- **需要内容补充的不代写**（占位未填、缺挂起原因、最终方案只有结论句）：列出清单交用户决策——方案内容是讨论产物，替用户编造就失去 RFC 的意义。
- 最后给一页汇总：查了几个、修复了什么、还剩什么待办。
