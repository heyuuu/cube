/**
 * 导入别名规约:相对攀升恰好到达 src 根的导入改写为 '@/…'。
 *   src/lib/a.ts        的 '../b'        → '@/b'        (攀升数 == 文件深度)
 *   src/lib/inner/a.ts  的 '../b'        不动           (解析仍落在 lib/ 内)
 *   src/lib/inner/a.ts  的 '../../b'     → '@/b'
 * 攀迁越过 src 根的路径是非法导入,留给 tsgo 报错,本规则不管。
 *
 * 说明符区间取自 TS AST(ImportDeclaration / ExportDeclaration / 动态 import())——
 * 字符串/注释/JSX 文本里形如 `from '…'` 的内容不构成导入(历史版本用正则扫裸文本,
 * 存在改写字符串内容的误伤面)
 */

import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { join, posix } from 'node:path';

import ts from 'typescript';

import type { LintRule } from './types.ts';

const ROOT = 'src';
// vendored/生成物随上游或生成器,不参与规约
const SKIP_PREFIXES = ['src/components/ui', 'src/components/liquikit'];

function walk(dir: string, out: string[] = []): string[] {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) {
      walk(full, out);
    } else if (/\.(ts|tsx|mts|cts)$/.test(name) && !name.endsWith('.d.ts')) {
      out.push(full);
    }
  }
  return out;
}

/** 一次替换:说明符去引号内容的原文区间与新文本 */
interface Edit {
  from: number;
  to: number;
  text: string;
}

/** 收集全部模块说明符字面量(static import / export…from / 动态 import()) */
function specifierEdits(sf: ts.SourceFile, depth: number): { edits: Edit[]; hits: number } {
  const edits: Edit[] = [];
  let hits = 0;

  const visit = (node: ts.Node) => {
    let lit: ts.StringLiteral | undefined;
    if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) {
      const spec = node.moduleSpecifier;
      if (spec && ts.isStringLiteral(spec)) {
        lit = spec;
      }
    } else if (
      ts.isCallExpression(node) &&
      node.expression.kind === ts.SyntaxKind.ImportKeyword &&
      ts.isStringLiteral(node.arguments[0])
    ) {
      lit = node.arguments[0];
    }
    if (lit) {
      const spec = lit.text;
      const climb = (spec.match(/^(\.\.\/)+/)?.[0].length ?? 0) / 3; // '../' 长度 3
      if (climb > 0 && climb === depth) {
        hits += 1;
        // 去引号内容区间 = 字面量起止各退一位
        edits.push({ from: lit.getStart(sf) + 1, to: lit.getEnd() - 1, text: `@/${spec.slice(climb * 3)}` });
      }
    }
    node.forEachChild(visit);
  };
  visit(sf);
  return { edits, hits };
}

export const checkImports: LintRule = {
  name: 'check-imports',
  run(fix: boolean): number {
    const files = walk(ROOT).filter((f) => !SKIP_PREFIXES.some((p) => posix.normalize(f).startsWith(p)));
    let total = 0;

    for (const file of files) {
      const depth = posix.normalize(file).split('/').length - 2; // src 起算的目录深度
      const source = readFileSync(file, 'utf8');
      const kind = file.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
      const sf = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, kind);
      const { edits, hits } = specifierEdits(sf, depth);

      if (hits > 0) {
        total += hits;
        console.log(`${fix ? 'fixed' : 'violation'}: ${file}(${hits} 处)`);
        if (fix) {
          // edits 按遍历序(位置升序)收集,从后往前套用保证偏移不失效
          let out = source;
          for (const e of edits.reverse()) {
            out = out.slice(0, e.from) + e.text + out.slice(e.to);
          }
          writeFileSync(file, out);
        }
      }
    }
    return total;
  },
};
