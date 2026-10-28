/**
 * 自定义 lint 规则统一入口:node scripts/lint.ts [--fix]。
 * 新增规则 = lint-rules/ 下实现 LintRule 并加入 RULES,由 pnpm check 统一驱动。
 */

import { blockComment } from './lint-rules/block-comment.ts';
import { checkImports } from './lint-rules/check-imports.ts';

const RULES = [checkImports, blockComment];
const FIX = process.argv.includes('--fix');

let violations = 0;
for (const rule of RULES) {
  violations += rule.run(FIX);
}

if (violations === 0) {
  console.log('[lint] 全部规则通过');
} else if (FIX) {
  console.log(`[lint] 已修复 ${violations} 处`);
} else {
  console.error(`[lint] ${violations} 处违例(--fix 自动修复)`);
  process.exitCode = 1;
}
