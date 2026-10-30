/**
 * 新建页面脚手架:tsx scripts/new-page.ts <pageName>(或 pnpm new-page <pageName>)。
 * 在 src/pages/<pageName>/ 下生成两个文件:
 *   <PageName>Page.tsx  空组件,导出 <PageName>Page(PageName 为 pageName 首字母大写)
 *   index.ts            入口,export * from './<PageName>Page'
 */

import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

// 相对仓库根,与 lint.ts 的 ROOT 同理,须从根目录运行
const PAGES_DIR = 'src/pages';

const pageName = process.argv[2];

if (!pageName) {
  console.error('用法: tsx scripts/new-page.ts <pageName>');
  process.exit(1);
}

// 组件名 = 首字母大写(pageName) + Page,页面名须保证它是合法标识符
if (!/^[a-z][A-Za-z0-9]*$/.test(pageName)) {
  console.error(`[new-page] 非法页面名 ${pageName}:须匹配 /^[a-z][A-Za-z0-9]*$/`);
  process.exit(1);
}

const pageDir = join(PAGES_DIR, pageName);
if (existsSync(pageDir)) {
  console.error(`[new-page] 目录已存在: src/pages/${pageName}/`);
  process.exit(1);
}

const componentName = pageName[0].toUpperCase() + pageName.slice(1) + 'Page';

mkdirSync(pageDir);
writeFileSync(
  join(pageDir, `${componentName}.tsx`),
  `export function ${componentName}() {
  return <div>新 ${componentName} 页, 待修改内容</div>;
}
`,
);
writeFileSync(join(pageDir, 'index.ts'), `export * from './${componentName}';\n`);

console.log(`[new-page] 已创建 src/pages/${pageName}/${componentName}.tsx`);
console.log(`[new-page] 已创建 src/pages/${pageName}/index.ts`);
