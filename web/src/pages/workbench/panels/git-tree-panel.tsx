import { useState } from 'react';

import { ErrorBanner } from '@/components/error-banner';
import { useWorkbenchInfo, useWorkbenchWorktrees } from '@/queries/workbench';
import type { WorktreeStatus } from '@/queries/workbench';

import { useWorktreeVisibility } from '../worktree-visibility';
import { CommitGraphSection } from './commit-graph';
import { WorktreeSection } from './worktree-section';
import { BranchAddDialog, BranchDeleteDialog, WorktreeAddDialog, WorktreeRemoveDialog } from './worktree-write';
import type { WorkbenchParams } from '../params';

export function GitTreePanel({ params }: { params: WorkbenchParams }) {
  const { path } = params;
  const info = useWorkbenchInfo(path);
  // 点击分支的定位信号：即使重复点同一分支（选中值不变）也要重新定位+闪烁
  const [focusTick, setFocusTick] = useState(0);
  // 工作副本显隐开关（默认全展示）：关闭的副本不注入 commit 图——dirty 的虚拟节点、
  // clean 的 HEAD 行装饰都不显示。纯视图过滤，不进 URL、不影响选中态；
  // 持久化按 path 隔离存 localStorage（见 worktree-visibility.ts）
  const { hiddenWorktrees, toggleWorktree } = useWorktreeVisibility(path);

  // 写侧对话框（1031）：面板内多个入口（副本区 + / 分支行 / commit 行）共用
  const [addPrefill, setAddPrefill] = useState<{ branch?: string; commitish?: string } | null>(null);
  const [removeTarget, setRemoveTarget] = useState<WorktreeStatus | null>(null);
  const [deleteBranchName, setDeleteBranchName] = useState<string | null>(null);
  const [branchAddOpen, setBranchAddOpen] = useState(false);
  const worktreesForDialogs = useWorkbenchWorktrees(path);
  const mainPath = worktreesForDialogs.data?.[0]?.path ?? path; // 副本列表主目录在前

  if (info.isPending) {
    return <div className="text-muted-foreground p-3 text-xs">加载中…</div>;
  }
  if (info.isError) {
    return <ErrorBanner message={info.error.message} />;
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="border-border shrink-0 overflow-y-auto border-b">
        <WorktreeSection
          path={path}
          params={params}
          onBranchPicked={() => setFocusTick((n) => n + 1)}
          hiddenWorktrees={hiddenWorktrees}
          onToggleWorktree={toggleWorktree}
          onAddWorktree={setAddPrefill}
          onRemoveWorktree={setRemoveTarget}
          onDeleteBranch={(name) => setDeleteBranchName(name)}
          onAddBranch={() => setBranchAddOpen(true)}
        />
      </div>
      <CommitGraphSection
        path={path}
        params={params}
        focusTick={focusTick}
        hiddenWorktrees={hiddenWorktrees}
        onAddWorktree={setAddPrefill}
      />
      {addPrefill ? <WorktreeAddDialog path={path} prefill={addPrefill} onClose={() => setAddPrefill(null)} /> : null}
      {removeTarget ? (
        <WorktreeRemoveDialog path={path} wt={removeTarget} mainPath={mainPath} onClose={() => setRemoveTarget(null)} />
      ) : null}
      {deleteBranchName ? (
        <BranchDeleteDialog path={path} branch={deleteBranchName} onClose={() => setDeleteBranchName(null)} />
      ) : null}
      {branchAddOpen ? <BranchAddDialog path={path} onClose={() => setBranchAddOpen(false)} /> : null}
    </div>
  );
}
