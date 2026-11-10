/**
 * 块注释形态规约:多行块注释(文档型与普通型)的开始/结束标识各自独立成行,
 * 内容行以 ` * ` 前缀书写;
 * 单行注释保持单行闭合,开始/结束标识与内容同行。
 *
 * 注释区间必须来自词法层真实判定——ts/tsx 走 TS AST 的 comment trivia,
 * css 走跳过引号字符串的专用扫描。字符串字面量 / JSX 文本 / 行注释里的 `/*`
 * 不构成块注释(历史版本按裸文本 indexOf 扫描,曾把 JSX 文案里的 `/*` 当注释
 * 开头改坏,且与 oxfmt 的 JSX 文本折叠互相拉锯,每跑一轮多出一个 ` * `)。
 */

import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { join, posix } from 'node:path';

import ts from 'typescript';

import type { LintRule } from './types.ts';

const ROOT = 'src';
// vendored/生成物随上游或生成器,不参与规约
const SKIP_PREFIXES = ['src/components/ui', 'src/components/liquikit', 'src/api/bindings.ts'];

function walk(dir: string, out: string[] = []): string[] {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) {
      walk(full, out);
    } else if (/\.(ts|tsx|mts|cts|css)$/.test(name) && !name.endsWith('.d.ts')) {
      out.push(full);
    }
  }
  return out;
}

/** 真实注释区间:start 指向开启标识,end 指向闭合标识之后 */
interface CommentSpan {
  start: number;
  end: number;
}

/**
 * ts/tsx:遍历 AST 节点的 leading/trailing trivia 收集块注释区间。
 * 解析器只把真正的注释暴露为 trivia——JSX 文本、字符串、行注释里的 `/*`
 * 根本不会出现在结果里,这是与裸文本扫描的本质区别
 */
function tsCommentSpans(file: string, source: string): CommentSpan[] {
  const kind = file.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const sf = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, kind);
  const out: CommentSpan[] = [];
  const seen = new Set<number>();
  const collect = (ranges: ts.CommentRange[] | undefined) => {
    for (const r of ranges ?? []) {
      if (source.startsWith('/*', r.pos) && !seen.has(r.pos)) {
        seen.add(r.pos);
        out.push({ start: r.pos, end: r.end });
      }
    }
  };
  const visit = (node: ts.Node) => {
    collect(ts.getLeadingCommentRanges(source, node.pos));
    collect(ts.getTrailingCommentRanges(source, node.end));
    node.forEachChild(visit);
  };
  visit(sf);
  // 文件末尾的注释挂在 EOF token 前,不在子节点遍历路径上
  collect(ts.getLeadingCommentRanges(source, sf.endOfFileToken.pos));
  return out.sort((a, b) => a.start - b.start);
}

/** css:无 AST 可用,手写扫描——只需跳过引号字符串(css 无正则/模板/JSX) */
function cssCommentSpans(source: string): CommentSpan[] {
  const out: CommentSpan[] = [];
  let i = 0;
  while (i < source.length) {
    const c = source[i];
    if (c === '"' || c === "'") {
      const quote = c;
      i += 1;
      while (i < source.length && source[i] !== quote) {
        i += source[i] === '\\' ? 2 : 1;
      }
      i += 1;
      continue;
    }
    if (c === '/' && source[i + 1] === '*') {
      const close = source.indexOf('*/', i + 2);
      const end = close < 0 ? source.length : close + 2;
      out.push({ start: i, end });
      i = end;
      continue;
    }
    i += 1;
  }
  return out;
}

/** 一次替换:覆盖注释首尾所在整行,from/to 均为原文区间 */
interface Edit {
  from: number;
  to: number;
  text: string;
}

/**
 * 单个多行块注释的整形:开启行标识后有内容则拆行(内容转 ` * ` 前缀,JSX 的 `{`
 * 并入开启行)、闭合行闭合标识前有内容则拆行(其后代码原样保留);
 * 中间内容行不规约。合规返回 null
 */
function reformat(source: string, span: CommentSpan): Edit | null {
  const { start, end } = span;
  const nl1 = source.indexOf('\n', start);
  if (nl1 < 0 || nl1 >= end) {
    return null; // 单行注释不规约
  }
  const token = source.startsWith('/**', start) ? '/**' : '/*';
  const openRest = source.slice(start + token.length, nl1);
  const nl2 = source.lastIndexOf('\n', end - 1);
  const closeBefore = source.slice(nl2 + 1, end - 2);
  if (openRest.trim() === '' && closeBefore.trim() === '') {
    return null;
  }

  const lineStart = source.lastIndexOf('\n', start - 1) + 1;
  const lineEndIdx = source.indexOf('\n', end);
  const lineEnd = lineEndIdx < 0 ? source.length : lineEndIdx;

  const prefix = source.slice(lineStart, start);
  const brace = prefix.endsWith('{');
  const indent = prefix.match(/^\s*/)?.[0] ?? '';
  const codeBefore = prefix.slice(0, brace ? -1 : undefined).trimEnd();
  const openContent = openRest.trim();
  const middle = source.slice(nl1 + 1, nl2 + 1); // 各中间行原文(含行尾换行),不规约
  const indentClose = source.slice(nl2 + 1, end - 2).match(/^\s*/)?.[0] ?? '';
  const afterClose = source.slice(end, lineEnd);

  const head: string[] = [];
  if (codeBefore !== '') {
    head.push(codeBefore);
  }
  head.push(indent + (brace ? '{' : '') + token);
  if (openContent !== '') {
    head.push(indent + ' * ' + openContent);
  }
  const tail: string[] = [];
  if (closeBefore.trim() !== '') {
    tail.push(closeBefore.trimEnd());
  }
  tail.push(indentClose + '*/' + afterClose);

  return { from: lineStart, to: lineEnd, text: head.join('\n') + '\n' + middle + tail.join('\n') };
}

export const blockComment: LintRule = {
  name: 'block-comment-close',
  run(fix: boolean): number {
    const files = walk(ROOT).filter((f) => !SKIP_PREFIXES.some((p) => posix.normalize(f).startsWith(p)));
    let total = 0;

    for (const file of files) {
      const source = readFileSync(file, 'utf8');
      const spans = file.endsWith('.css') ? cssCommentSpans(source) : tsCommentSpans(file, source);

      let violations = 0;
      const edits: Edit[] = [];
      // 同一行挤着多个待整形注释时区间会重叠,只修最靠后的一个,其余下一轮收敛
      let guard = source.length;
      for (let i = spans.length - 1; i >= 0; i--) {
        const e = reformat(source, spans[i]);
        if (!e) {
          continue;
        }
        violations += 1;
        if (e.to <= guard) {
          edits.push(e);
          guard = e.from;
        }
      }
      if (violations === 0) {
        continue;
      }

      total += violations;
      console.log(`${fix ? 'fixed' : 'violation'}: ${file}(${violations} 处)`);
      if (fix) {
        // edits 按区间降序收集,从后往前套用保证前面的偏移不失效
        let out = source;
        for (const e of edits) {
          out = out.slice(0, e.from) + e.text + out.slice(e.to);
        }
        writeFileSync(file, out);
      }
    }
    return total;
  },
};
