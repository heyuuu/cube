#!/usr/bin/env python3
"""校验 docs/rfcs 下 RFC 文档的机械一致性。

用法：
    python3 check.py docs/rfcs                  # 校验全部 RFC
    python3 check.py docs/rfcs/rfc-005-目录对比页  # 只校验单个 RFC 目录

规则来源 docs/rfcs/README.md；退出码：发现错误为 1，仅警告或全部通过为 0。
"""

import re
import sys
from pathlib import Path

# 合法状态，与 docs/rfcs/README.md 状态机保持同步
STATUSES = ("draft", "discussion", "accepted", "developing", "done", "parked", "rejected")
# 模板内置的六个必需章节
SECTIONS = ("背景与目标", "需求描述", "方案讨论", "最终方案", "实现记录", "遗留问题")
FRONT_FIELDS = ("id", "title", "status", "created", "updated")

DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
DIR_RE = re.compile(r"^rfc-(\d{3})-\S+$")


class Result:
    def __init__(self, name):
        self.name = name
        self.errors = []
        self.warnings = []

    def error(self, msg):
        self.errors.append(msg)

    def warn(self, msg):
        self.warnings.append(msg)


def parse_front_matter(text):
    """返回 (front-matter 字典, 正文)；无 front-matter 时返回 (None, 全文)。"""
    if not text.startswith("---\n"):
        return None, text
    end = text.find("\n---\n", 4)
    if end < 0:
        return None, text
    fm = {}
    for line in text[4:end].splitlines():
        key, sep, value = line.partition(":")
        if sep:
            fm[key.strip()] = value.strip()
    return fm, text[end + 5 :]


def section_text(body, name):
    """取章节正文；章节不存在返回 None。"""
    m = re.search(rf"^## {re.escape(name)}\s*$", body, re.M)
    if not m:
        return None
    rest = body[m.end() :]
    nxt = re.search(r"^## ", rest, re.M)
    return rest[: nxt.start()] if nxt else rest


def is_placeholder(section):
    """剔除空行与模板引导语（> 引用行）后没有实质内容 → 仍是占位。"""
    lines = [l for l in section.splitlines() if l.strip() and not l.strip().startswith(">")]
    return not lines


def filled(body, name):
    sect = section_text(body, name)
    return sect is not None and not is_placeholder(sect)


def check_rfc(dir_path):
    num = DIR_RE.match(dir_path.name).group(1)
    res = Result(dir_path.name)

    rfc_path = dir_path / "RFC.md"
    if not rfc_path.exists():
        res.error("缺少 RFC.md（编号目录内唯一必须文件）")
        return res

    text = rfc_path.read_text(encoding="utf-8")
    fm, body = parse_front_matter(text)
    if fm is None:
        res.error("缺少 YAML front-matter（文件须以 --- 元信息块开头）")
        return res

    for field in FRONT_FIELDS:
        if not fm.get(field):
            res.error(f"front-matter 缺少字段 {field}")

    if fm.get("id") and fm["id"] != num:
        res.error(f"id（{fm['id']}）与目录编号（{num}）不一致")

    status = fm.get("status", "")
    if status and status not in STATUSES:
        res.error(f"status 非法：{status}（合法值：{' / '.join(STATUSES)}）")

    created, updated = fm.get("created", ""), fm.get("updated", "")
    for label, value in (("created", created), ("updated", updated)):
        if value and not DATE_RE.match(value):
            res.error(f"{label} 日期格式非法：{value}（应为 YYYY-MM-DD）")
    if DATE_RE.match(created) and DATE_RE.match(updated) and updated < created:
        res.error(f"updated（{updated}）早于 created（{created}）")

    title = fm.get("title", "")
    h1 = re.search(r"^# (.+)$", body, re.M)
    if title and not h1:
        res.error("正文缺少 H1 标题（# 需求名）")
    elif title and h1 and h1.group(1).strip() != title:
        res.error(f"H1（{h1.group(1).strip()}）与 title（{title}）不一致")

    missing = [s for s in SECTIONS if section_text(body, s) is None]
    if missing:
        res.error(f"缺少章节：{'、'.join(missing)}")

    # 占位检查：状态推进一步，对应章节就应脱离模板占位
    if status in ("accepted", "developing", "done") and not filled(body, "最终方案"):
        res.error(f"status 已是 {status}，但「最终方案」仍是模板占位")
    if status == "developing" and not filled(body, "实现记录"):
        res.warn("developing 中「实现记录」尚未填写（有实现产出后回写）")
    if status == "done":
        if not filled(body, "实现记录"):
            res.error("done 但「实现记录」仍是占位")
        if not filled(body, "遗留问题"):
            res.warn("done 但「遗留问题」仍是占位（没有遗留也应写明「无」）")
        unchecked = body.count("- [ ]")
        if unchecked:
            res.warn(f"done 但有 {unchecked} 条验收标准未勾选")

    return res


def check_root(root):
    """校验整个 rfcs 目录，返回 Result 列表。"""
    dirs = sorted(d for d in root.iterdir() if d.is_dir() and d.name != "_template")
    results, numbers = [], []
    for d in dirs:
        if DIR_RE.match(d.name):
            results.append(check_rfc(d))
            numbers.append(int(DIR_RE.match(d.name).group(1)))
        else:
            r = Result(d.name)
            r.error("目录名不符合 rfc-{三位编号}-{需求名}")
            results.append(r)

    dup = sorted({n for n in numbers if numbers.count(n) > 1})
    if dup:
        print(f"错误: 编号重复：{', '.join(f'{n:03d}' for n in dup)}")
    if numbers:
        if min(numbers) != 1:
            print(f"警告: 最小编号是 {min(numbers):03d}，编号应从 001 开始")
        gaps = [n for n in range(1, max(numbers) + 1) if n not in numbers]
        if gaps:
            print(f"警告: 编号跳号缺失：{', '.join(f'{n:03d}' for n in gaps)}（编号永不复用，跳号需确认）")

    if not (root / "_template" / "RFC.md").exists():
        print("警告: 缺少 _template/RFC.md，无法新建 RFC")

    return results


def main():
    target = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("docs/rfcs")
    if not target.is_dir():
        print(f"目录不存在：{target}")
        return 2

    if DIR_RE.match(target.name):
        results = [check_rfc(target)]
    else:
        results = check_root(target)

    err_total = warn_total = 0
    for r in results:
        if not r.errors and not r.warnings:
            continue
        print(f"== {r.name} ==")
        for e in r.errors:
            print(f"  错误: {e}")
        for w in r.warnings:
            print(f"  警告: {w}")
        err_total += len(r.errors)
        warn_total += len(r.warnings)

    if err_total == 0 and warn_total == 0:
        print(f"校验通过：{len(results)} 个 RFC 无问题")
        return 0
    print(f"汇总：{len(results)} 个 RFC，{err_total} 错误，{warn_total} 警告")
    return 1 if err_total else 0


if __name__ == "__main__":
    sys.exit(main())
