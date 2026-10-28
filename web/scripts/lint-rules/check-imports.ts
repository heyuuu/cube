/**
 * 导入别名规约:相对攀升恰好到达 src 根的导入改写为 '@/…'。
 *   src/lib/a.ts        的 '../b'        → '@/b'        (攀升数 == 文件深度)
 *   src/lib/inner/a.ts  的 '../b'        不动           (解析仍落在 lib/ 内)
 *   src/lib/inner/a.ts  的 '../../b'     → '@/b'
 * 攀迁越过 src 根的路径是非法导入,留给 tsgo 报错,本规则不管。
 */

import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { join, posix } from 'node:path';

import type { LintRule } from './types.ts';

const ROOT = 'src';
// vendored/生成物随上游或生成器,不参与规约
const SKIP_PREFIXES = ['src/components/ui', 'src/components/liquikit'];
// 覆盖 static import/from、动态 import()、export … from
const SPEC_RE = /(?:\bfrom\s+|\bimport\s*\(\s*|\bimport\s+)['"]([^'"]+)['"]/g;

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

export const checkImports: LintRule = {
  name: 'check-imports',
  run(fix: boolean): number {
    const files = walk(ROOT).filter((f) => !SKIP_PREFIXES.some((p) => posix.normalize(f).startsWith(p)));
    let total = 0;

    for (const file of files) {
      const depth = posix.normalize(file).split('/').length - 2; // src 起算的目录深度
      let hits = 0;
      const source = readFileSync(file, 'utf8');
      const rewritten = source.replace(SPEC_RE, (match, spec: string) => {
        const climb = (spec.match(/^(\.\.\/)+/)?.[0].length ?? 0) / 3; // '../' 长度 3
        if (climb === 0 || climb !== depth) {
          return match;
        }
        hits += 1;
        return match.replace(spec, `@/${spec.slice(climb * 3)}`);
      });
      if (hits > 0) {
        total += hits;
        console.log(`${fix ? 'fixed' : 'violation'}: ${file}(${hits} 处)`);
        if (fix) {
          writeFileSync(file, rewritten);
        }
      }
    }
    return total;
  },
};
