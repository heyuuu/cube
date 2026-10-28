import { defineConfig } from 'oxlint';

export default defineConfig({
  plugins: [
    // oxlint 默认项
    'eslint',
    'typescript',
    'unicorn',
    'oxc',
    // 库基础
    'import',
    // 前端
    'react',
    'react-perf',
    // 测试
    'vitest',
  ],
  rules: {
    // if/for/while 的 body 必须带花括号
    curly: ['error', 'all'],

    // import 语句内排序
    'sort-imports': [
      'warn',
      {
        ignoreDeclarationSort: true, // 忽略 import 语句间排序(语句间排序是 oxfmt 的工作)
        ignoreCase: true,
      },
    ],

    /* import */
    // 禁止循环导入
    'import/no-cycle': 'error',
    // 将同源 import 合并(不合并 import {} 和 import type {})
    'import/no-duplicates': 'warn',
    // 将 import type 和 import value 彻底分开
    'import/consistent-type-specifier-style': ['warn', 'prefer-top-level'],
    // 项目内 import 一律不带 .ts/.tsx 后缀（vite/bundler 解析无碍，与 tsc 侧关闭
    // allowImportingTsExtensions 配套）；css 等资源导入必须带扩展名，不在约束内
    'import/extensions': ['error', 'ignorePackages', { ts: 'never', tsx: 'never' }],

    /* react */
    //
    'react/rules-of-hooks': 'error',
    // off：hook / 常量 / 工具与组件同文件导出常态，
    // 代价仅是编辑时 fast refresh 粒度变粗，不值得拆文件；
    // shadcn vendored 组件（src/components/ui）同理。
    'react/only-export-components': 'off',

    /* vitest */
    // 拓扑不变量校验收敛在 verifyGraphInvariants 内部断言，告诉规则它是断言函数
    'vitest/expect-expect': ['warn', { assertFunctionNames: ['expect', 'verifyGraphInvariants'] }],
  },
  // 不进行 lint 的文件 (比如代码生成文件)
  ignorePatterns: [],
  // 具体文件规则覆盖
  overrides: [
    {
      // shadcn 为第三方源码，有些不符合要求的可以在这里跳过，方便随上游更新(但是最好别直接ignore)
      files: ['src/components/ui/**'],
      rules: {},
    },
  ],
});
