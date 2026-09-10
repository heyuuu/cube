import { defineConfig } from 'oxlint';

export default defineConfig({
  plugins: ['react', 'typescript', 'oxc'],
  rules: {
    'react/react-compiler': 'error',
    'react/rules-of-hooks': 'error',
    // off：hook / 常量 / 工具与组件同文件导出是本仓库刻意布局（如 useTreePanePrefs、
    // md 主题表、previewKindOf），代价仅是编辑时 fast refresh 粒度变粗，不值得拆文件；
    // shadcn vendored 组件（src/components/ui）同理。
    'react/only-export-components': 'off',
  },
});
