# cube

基于 React 19 + Vite + TypeScript 的单页应用模板，UI 使用 shadcn（Base UI 风格）+ Tailwind CSS v4，路由使用 react-router，服务端状态使用 TanStack Query。

## 技术栈

- [React 19](https://react.dev/) + [React Compiler](https://react.dev/learn/react-compiler)（组件自动 memo 化，无需手写 useMemo / useCallback）
- [Vite 8](https://vite.dev/) + [TypeScript](https://www.typescriptlang.org/)（typecheck 使用 tsgo）
- [Tailwind CSS v4](https://tailwindcss.com/)（CSS-first 配置，见 `src/index.css`，无 tailwind.config）
- [shadcn](https://ui.shadcn.com/) 组件（底层为 [Base UI](https://base-ui.com/)，vendored 在 `src/components/ui/`）
- [react-router](https://reactrouter.com/)（路由表集中在 `src/App.tsx`）
- [@tanstack/react-query](https://tanstack.com/query)（全局 QueryClient 在 `src/lib/query.ts`，DevTools 仅开发环境挂载）
- [oxlint](https://oxc.rs/docs/guide/usage/linter) + [oxfmt](https://github.com/oxc-project/oxfmt) + 自定义 lint 规则（`scripts/lint.ts`）
- [Vitest](https://vitest.dev/) 测试

## 快速开始

```bash
pnpm install
pnpm dev
```

## 常用命令

| 命令                                             | 说明                                          |
| ------------------------------------------------ | --------------------------------------------- |
| `pnpm dev`                                       | 启动开发服务器                                |
| `pnpm build`                                     | 类型检查 + 生产构建（`tsc -b && vite build`） |
| `pnpm preview`                                   | 预览生产构建                                  |
| `pnpm check`                                     | 提交前总闸：lint + typecheck + format 检查    |
| `pnpm fix`                                       | 自动修复 lint 与格式问题                      |
| `pnpm lint` / `pnpm lint:fix`                    | oxlint + 自定义规则                           |
| `pnpm typecheck`                                 | TypeScript 类型检查（tsgo）                   |
| `pnpm format` / `pnpm format:check`              | oxfmt 格式化                                  |
| `pnpm test` / `pnpm test:watch` / `pnpm test:ui` | 运行测试（收集 `src/**/*.test.ts`）           |
| `pnpm new-page <name>`                           | 生成新页面脚手架                              |

注意：脚本内部按 `src/` 相对路径查找文件，请从仓库根目录运行。

## 目录结构

```
src/
├── App.tsx            # 路由表（单一数据源，带 nav 的路由自动进侧边栏和面包屑）
├── main.tsx           # 入口，渲染前应用主题以防首帧闪烁
├── index.css          # Tailwind v4 与主题变量
├── pages/             # 每页一个目录
│   ├── index/         #   IndexPage.tsx + index.ts
│   ├── settings/
│   ├── single/        # 不带布局壳的独立页面示例
│   └── layout/        # 侧边栏布局（Sidebar + Breadcrumb，子路由渲染在 <Outlet />）
├── components/ui/     # shadcn 组件（vendored，尽量别手改）
├── queries/           # 数据请求
├── hooks/             # use-theme、use-mobile（768px 断点）
└── lib/               # query.ts（全局 QueryClient）、utils.ts（cn 等）
```

## 如何添加新页面

```bash
pnpm new-page my-page
```

页面名须匹配 `/^[a-z][A-Za-z0-9]*$/`。然后在 `src/App.tsx` 的 `pageRoutes` 中注册路由；带 `nav` 字段的路由会自动出现在侧边栏和面包屑中，不需要在别处再配置一遍。

## 添加 shadcn 组件

```bash
pnpm exec shadcn add dialog
```

组件会生成到 `src/components/ui/`，并被自定义 lint 规则豁免。

## 主题

暗色模式通过 `<html>` 上的 `.dark` class 实现，`useTheme()` 挂在 App 根上，可用 Cmd/Ctrl+D 快速切换。首帧主题在 `main.tsx` 渲染前应用，避免闪烁。
