/**
 * 块注释形态规约:多行块注释(文档型与普通型)的开始/结束标识各自独立成行,
 * 内容行以 ` * ` 前缀书写;
 * 单行注释保持单行闭合,开始/结束标识与内容同行。
 */

import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { join, posix } from 'node:path';

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

/** 行内块注释扫描:opens = 有未闭合的 `/*`,closes = 出现过成对开闭 */
function scanLine(line: string): { opens: boolean; closes: boolean } {
  let opens = false;
  let closes = false;
  let i = line.indexOf('/*');
  while (i >= 0) {
    const close = line.indexOf('*/', i + 2);
    if (close >= 0) {
      closes = true;
      i = line.indexOf('/*', close + 2);
    } else {
      opens = true;
      break;
    }
  }
  return { opens, closes };
}

/**
 * 截掉行注释前缀之后的 `//` 部分:`//` 里引用的块注释 token(如被行注释
 * 禁用的示例注释)不构成真块注释;
 * 字符串内的 `//`(如 'http://…')会连带截断,属可忽略的启发式误伤
 */
function stripLineComment(line: string): string {
  const lc = line.indexOf('//');
  return lc >= 0 ? line.slice(0, lc) : line;
}

export const blockComment: LintRule = {
  name: 'block-comment-close',
  run(fix: boolean): number {
    const files = walk(ROOT).filter((f) => !SKIP_PREFIXES.some((p) => posix.normalize(f).startsWith(p)));
    let total = 0;

    for (const file of files) {
      const lines = readFileSync(file, 'utf8').split('\n');
      let inComment = false;
      let hits = 0;
      const out: string[] = [];

      for (const line of lines) {
        if (!inComment) {
          // 判定基于截掉行注释后的内容,改写仍取原行(位置前缀一致)
          const effective = stripLineComment(line);
          const open = effective.indexOf('/*');
          if (open < 0 || !scanLine(effective).opens) {
            out.push(line);
            continue;
          }
          // 跨行注释的开启行:标识后有内容则拆出独立行,内容转 ` * ` 前缀;
          // JSX 表达式的 `{` 并入开启行(`{/*`),否则 `{` 悬空更怪
          const token = line.startsWith('/**', open) ? '/**' : '/*';
          const brace = open > 0 && line[open - 1] === '{';
          const rest = line.slice(open + token.length).trimStart();
          const indent = line.slice(0, open - (brace ? 1 : 0)).match(/^\s*/)?.[0] ?? '';
          const codeBefore = line.slice(0, open - (brace ? 1 : 0)).trimEnd();
          if (rest === '') {
            out.push(line);
            inComment = true;
            continue;
          }
          hits += 1;
          if (codeBefore !== '') {
            out.push(codeBefore);
          }
          out.push(indent + (brace ? '{' : '') + token, `${indent} * ${rest}`);
          inComment = true;
          continue;
        }

        const close = line.indexOf('*/');
        if (close < 0) {
          out.push(line);
          continue;
        }
        // 闭合行:`*/` 之前的尾部内容须为空(其后内容如 JSX 的 `}` 原样保留)
        const before = line.slice(0, close).trimEnd();
        if (before.trim() === '') {
          inComment = scanLine(line.slice(close + 2)).opens;
          out.push(line);
          continue;
        }
        hits += 1;
        const indent = line.match(/^\s*/)?.[0] ?? '';
        const after = line.slice(close + 2);
        out.push(before, `${indent}*/${after}`);
        inComment = scanLine(after).opens;
      }

      if (hits > 0) {
        total += hits;
        console.log(`${fix ? 'fixed' : 'violation'}: ${file}(${hits} 处)`);
        if (fix) {
          writeFileSync(file, out.join('\n'));
        }
      }
    }
    return total;
  },
};
