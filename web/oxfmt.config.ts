import { defineConfig } from 'oxfmt';

export default defineConfig({
  printWidth: 120,
  singleQuote: true,
  // tailwindow 类名排序
  sortTailwindcss: true,
  // sort imports
  sortImports: {
    groups: [
      // 内置包
      ['value-builtin'],
      { newlinesBetween: false },
      ['type-builtin'],
      // 第三方包
      ['value-external'],
      { newlinesBetween: false },
      ['type-external'],
      // 项目包
      ['value-internal', 'value-subpath'],
      { newlinesBetween: false },
      ['type-internal', 'type-subpath'],
      // 相邻包
      ['value-parent', 'value-sibling', 'value-index'],
      { newlinesBetween: false },
      ['type-parent', 'type-sibling', 'type-index'],
      // 其他
      'unknown',
      // 副作用包
      'side_effect',
      'side_effect_style',
    ],
  },
});
