import { useQueryClient } from '@tanstack/react-query';
// workspace 规则编辑弹窗（1030）：从项目详情抽屉直达（cube.json 是仓库级事实，
// 入口必须在项目上下文内，不放全局 settings）。轻量自绘 overlay（同 ConfirmDialog 模式）。
import { useState } from 'react';

import { DialogShell } from '@/components/dialog-shell';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useWorkspaceReset, useWorkspaceSave, useWorkspaceState } from '@/queries/project';
import type { Project } from '@/api/client';

type Rule = { name: string; path: string };

// 特殊规则快捷入口（path 读标准 monorepo 声明文件展开）——替代旧「导入探测候选」：
// 要 npm/pnpm 探测结果就声明对应特殊规则，裁剪交给 glob/精确路径
const QUICK_RULES = [
  { label: '+ <pnpm>', path: '<pnpm>', hint: '读 pnpm-workspace.yaml 的 packages' },
  { label: '+ <npm>', path: '<npm>', hint: '读 package.json 的 workspaces' },
];

const PATH_HINT = '相对路径 / glob / <npm>';

export function WorkspaceDialog({ project, onClose }: { project: Project; onClose: () => void }) {
  const qc = useQueryClient();
  const state = useWorkspaceState(project.path);
  const save = useWorkspaceSave();
  const reset = useWorkspaceReset();

  const [rules, setRules] = useState<Rule[]>([]);

  // 打开时以当前规则初始化（未声明时后端已填默认规则——编辑起点即当前生效配置）。
  // 数据异步到达，用「渲染期间调整 state」模式同步（React 推荐，优于 effect）
  const current = state.data?.rules;
  const [prevRules, setPrevRules] = useState(current);
  if (current !== prevRules) {
    setPrevRules(current);
    setRules((current ?? []).map((r) => ({ ...r })));
  }

  function setAt(i: number, patch: Partial<Rule>) {
    setRules((prev) => prev.map((m, j) => (j === i ? { ...m, ...patch } : m)));
  }

  function addRule(path = '') {
    setRules((prev) => [...prev, { name: '', path }]);
  }

  async function invalidate() {
    await qc.invalidateQueries({ queryKey: ['project', 'list'] });
    await qc.invalidateQueries({ queryKey: ['project', 'workspace', project.path] });
  }

  function submit() {
    save.mutate(
      { path: project.path, workspaces: rules },
      {
        onSuccess: async () => {
          await invalidate();
          onClose();
        },
      },
    );
  }

  function doReset() {
    reset.mutate(
      { path: project.path },
      {
        onSuccess: async () => {
          await invalidate();
          onClose();
        },
      },
    );
  }

  return (
    <DialogShell title="workspace 规则" onClose={onClose} className="flex max-h-[80vh] max-w-lg flex-col">
      <div className="text-muted-foreground mt-1 text-xs">
        写入项目内 .cube/cube.json（进 git）。path 支持相对目录、glob 通配（apps/**）与特殊规则
        {' <npm>/<pnpm>（读标准 monorepo 声明）'}；name 可选，通配/特殊规则作组前缀。清空保存 = 显式声明无 workspace。
      </div>

      {state.isLoading ? (
        <div className="text-muted-foreground py-6 text-center text-xs">加载中…</div>
      ) : (
        <>
          <div className="mt-3 flex-1 space-y-2 overflow-y-auto">
            {rules.length === 0 && (
              <div className="text-muted-foreground py-2 text-xs">
                暂无规则（{state.data?.declared ? '显式声明为空' : '未声明时默认 <npm>+<pnpm> 生效'}）
              </div>
            )}
            {rules.map((r, i) => (
              <div key={i} className="flex items-center gap-2">
                <Input
                  className="h-8 w-32 shrink-0"
                  value={r.name}
                  placeholder="名称(可选)"
                  onChange={(e) => setAt(i, { name: e.target.value })}
                />
                <Input
                  className="h-8 flex-1 font-mono text-xs"
                  value={r.path}
                  placeholder={PATH_HINT}
                  onChange={(e) => setAt(i, { path: e.target.value })}
                />
                <Button variant="ghost" size="sm" onClick={() => setRules((prev) => prev.filter((_, j) => j !== i))}>
                  移除
                </Button>
              </div>
            ))}
          </div>

          <div className="mt-3 flex flex-wrap items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => addRule()}>
              添加
            </Button>
            {QUICK_RULES.map((q) => (
              <Button
                key={q.path}
                variant="outline"
                size="sm"
                title={q.hint}
                className="font-mono"
                onClick={() => addRule(q.path)}
              >
                {q.label}
              </Button>
            ))}
            {state.data?.declared && (
              <Button
                variant="outline"
                size="sm"
                onClick={doReset}
                disabled={reset.isPending}
                title="删除 workspaces 节，回到默认 <npm>+<pnpm> 规则"
              >
                恢复默认
              </Button>
            )}
            {save.isError && <span className="text-destructive text-xs">{save.error.message}</span>}
          </div>

          <div className="mt-4 flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={onClose}>
              取消
            </Button>
            <Button size="sm" onClick={submit} disabled={save.isPending || rules.some((r) => !r.path)}>
              保存
            </Button>
          </div>
        </>
      )}
    </DialogShell>
  );
}
