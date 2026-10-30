# AGENTS.md

React 19 + Vite + TypeScript SPA。UI 为 shadcn（base-mira 风格，底层是 Base UI 而非 Radix）+ Tailwind CSS v4（CSS-first，配置在 `src/index.css`，无 tailwind.config）。路由用 react-router（声明式 `<Routes>`，路由表集中在 `src/App.tsx`），服务端状态用 @tanstack/react-query。包管理器为 pnpm。

## 常用命令

- `pnpm dev` — 开发服务器
- `pnpm check` — 提交前总闸：lint + typecheck + format:check
- `pnpm fix` — 自动修复 lint 与 format
- `pnpm lint` / `pnpm lint:fix` — oxlint + 自定义规则（`scripts/lint.ts`）
- `pnpm typecheck` — tsgo（TypeScript 原生预览版）
- `pnpm format` / `format:check` — oxfmt
- `pnpm build` — tsc -b && vite build
- `pnpm test` — vitest（node 环境，只收集 `src/**/*.test.ts`）
- `pnpm new-page <name>` — 新建页面脚手架

脚本内部使用相对路径（`lint.ts`、`new-page.ts` 均以 `src/` 起算），必须从仓库根目录运行。

## 目录结构

- `src/pages/<name>/` — 每页一个目录：`<Name>Page.tsx`（页面组件）+ `index.ts`（`export * from './<Name>Page'`），外部只从 `@/pages/<name>` 导入。新页面用 `pnpm new-page <name>` 生成，页面名须匹配 `/^[a-z][A-Za-z0-9]*$/`
- `src/pages/layout/` — 侧边栏布局壳（Sidebar + Breadcrumb），作为布局路由，子路由渲染在 `<Outlet />`
- `src/components/ui/` — shadcn vendored 组件，随上游生成，已豁免自定义 lint 规则；尽量别手改，新增组件用 `pnpm exec shadcn add <component>`
- `src/lib/query.ts` — 全局唯一 QueryClient（staleTime 30s、retry 1）；DevTools 仅 DEV 挂载
- `src/hooks/` — `use-theme`（主题）、`use-mobile`（768px 断点）
- `scripts/lint-rules/` — 自定义 lint 规则；新增规则 = 实现 `LintRule` 接口并加入 `lint.ts` 的 `RULES`

## 代码规约

多数有 linter/formatter 兜底，但别依赖 `--fix`：

- import：`@/` 别名指向 `src/`；相对路径攀升恰好到 src 根时必须改写为 `@/…`；项目内 import 一律不带 `.ts`/`.tsx` 后缀
- `import type` 与值导入彻底分开（顶层 type specifier，配合 verbatimModuleSyntax）；禁止循环导入
- `erasableSyntaxOnly`：禁用 enum、namespace、构造器参数属性；`noUnusedLocals` / `noUnusedParameters` 开启
- 多行块注释：`/*` 与 `*/` 各自独立成行，内容行以 `*` 前缀书写
- oxfmt：单引号、行宽 120、自动排序 import（内置 → 第三方 → 项目内 → 相邻目录，值与 type 相邻不空行）与 tailwind 类名
- hook / 常量 / 工具与组件同文件导出是允许的（`react/only-export-components` 已关），不必为此拆文件

## React Compiler 已启用

构建管线带 babel-plugin-react-compiler，组件自动 memo 化：不要手写 useMemo / useCallback / React.memo 做性能补偿；代价是必须严格遵守 Rules of Hooks（oxlint 已开 error）。

## 主题

暗色模式 = `<html>` 上的 `.dark` class。`useTheme()` 已挂在 App 根（Cmd/Ctrl+D 切换，壳外路由同样生效）；`applyThemeEarly()` 在 `main.tsx` 渲染前调用以防首帧闪烁，勿改动调用时机。
