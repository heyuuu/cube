// settings 页「模板源」分区：cube create 的 source 名 ↔ git 仓库地址映射
// （增删改 + 拖拽排序）。数据源 /api/template/source/list（settings.json tplSources 节），
// 保存即生效。交互模板沿用 Forge 分区：Sheet 抽屉编辑、删除前确认、grip 拖拽排序
// （顺序即 create 交互选择序）。
import { GripVertical } from 'lucide-react';
import { useState } from 'react';

import type { TplSource } from '@/api/client';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ErrorBanner } from '@/components/error-banner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useTplSourceDelete, useTplSourceReorder, useTplSourceSave, useTplSources } from '@/queries/template';

import { STICKY_LEFT, STICKY_RIGHT, useDragOrder } from './drag-order';

interface TplSourceDraft {
  name: string;
  repoUrl: string;
}

const EMPTY_DRAFT: TplSourceDraft = { name: '', repoUrl: '' };

// 编辑表单（右侧抽屉）：关闭即放弃草稿，保存成功后自动关闭
function TplSourceForm({ draft, onClose }: { draft: TplSourceDraft; onClose: () => void }) {
  const [form, setForm] = useState<TplSourceDraft>(draft);
  const save = useTplSourceSave();
  const del = useTplSourceDelete();
  const isEdit = draft.name !== '';
  const set = <K extends keyof TplSourceDraft>(key: K, value: TplSourceDraft[K]) =>
    setForm((f) => ({ ...f, [key]: value }));

  const submit = () => {
    const newName = form.name.trim();
    save.mutate(
      { name: newName, repoUrl: form.repoUrl.trim() },
      {
        onSuccess: () => {
          // name 是唯一键：编辑改了 name 等价于删旧存新，补一步删旧
          if (isEdit && newName !== draft.name) del.mutate({ name: draft.name });
          onClose();
        },
      },
    );
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent className="w-full gap-0 overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>{isEdit ? '编辑模板源' : '新增模板源'}</SheetTitle>
          <SheetDescription>保存即生效（settings.json），cube create 按 name 引用该模板仓库</SheetDescription>
        </SheetHeader>
        <div className="flex flex-col gap-4 p-4 text-sm">
          {save.error && <ErrorBanner message={`保存失败：${save.error.message}`} />}

          <label className="flex flex-col gap-1">
            <span className="text-xs text-muted-foreground">name（cube create 的引用短名；core 为默认源）</span>
            <Input
              value={form.name}
              onChange={(e) => set('name', e.target.value)}
              placeholder="core"
              className="font-mono"
            />
          </label>

          <label className="flex flex-col gap-1">
            <span className="text-xs text-muted-foreground">repoUrl（git 仓库地址）</span>
            <Input
              value={form.repoUrl}
              onChange={(e) => set('repoUrl', e.target.value)}
              placeholder="git@github.com:heyuuu/cube-templates.git"
              className="font-mono"
            />
          </label>

          <div className="mt-2 flex justify-end gap-2">
            <Button size="sm" variant="outline" onClick={onClose}>
              取消
            </Button>
            <Button size="sm" disabled={save.isPending || !form.name.trim() || !form.repoUrl.trim()} onClick={submit}>
              保存
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}

export function TplSourceSection() {
  const sources = useTplSources();
  const del = useTplSourceDelete();
  const reorder = useTplSourceReorder();
  const [editing, setEditing] = useState<TplSourceDraft | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);

  const list = sources.data?.list ?? [];
  const defaultSource = sources.data?.defaultSource ?? 'core';
  const d = useDragOrder(
    (s: TplSource) => s.name,
    list,
    (rows) => reorder.mutate({ names: rows.map((r) => r.name) }),
  );
  const hasDefault = list.some((s) => s.name === defaultSource);

  return (
    <section>
      <div className="mb-2 flex items-baseline gap-2">
        <h2 className="text-sm font-medium">模板源（cube create）</h2>
        <span className="text-xs text-muted-foreground">
          source 名 ↔ git 仓库地址映射；cube create 按 name 引用（@name/模板名），缺省 source 为 {defaultSource}；拖动 ⠿
          排序
        </span>
        <Button size="sm" variant="outline" className="ml-auto" onClick={() => setEditing(EMPTY_DRAFT)}>
          新增
        </Button>
      </div>
      {/* 默认源缺失提醒：不带 @ 的模板引用（cube create 模板名 目标路径）依赖它 */}
      {!sources.isPending && !sources.error && !hasDefault && (
        <p className="mb-2 text-xs text-amber-600 dark:text-amber-400">
          默认源 {defaultSource} 未配置：cube create 不带 @ 的模板引用将不可用
        </p>
      )}
      {sources.error && <ErrorBanner message={`加载失败：${sources.error.message}`} />}
      {del.error && <ErrorBanner message={`删除失败：${del.error.message}`} />}
      {reorder.error && <ErrorBanner message={`排序失败：${reorder.error.message}`} />}
      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className={STICKY_LEFT}>source</TableHead>
              <TableHead>repoUrl</TableHead>
              <TableHead className={`${STICKY_RIGHT} text-right`}>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {d.rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={3} className="text-xs text-muted-foreground">
                  暂无模板源配置
                </TableCell>
              </TableRow>
            )}
            {d.rows.map((s: TplSource, i) => (
              <TableRow key={s.name} {...d.rowProps(s, i)}>
                <TableCell className={`${STICKY_LEFT} font-medium`}>
                  <span className="flex items-center gap-1.5">
                    <GripVertical {...d.gripProps(s)} />
                    <span className="font-mono text-xs">{s.name}</span>
                    {s.name === defaultSource && (
                      <Badge variant="secondary" className="text-[10px]">
                        默认
                      </Badge>
                    )}
                  </span>
                </TableCell>
                <TableCell className="max-w-[36rem] truncate font-mono text-xs text-muted-foreground" title={s.repoUrl}>
                  {s.repoUrl}
                </TableCell>
                <TableCell className={`${STICKY_RIGHT} text-right`}>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() =>
                      setEditing({
                        name: s.name,
                        repoUrl: s.repoUrl,
                      })
                    }
                  >
                    编辑
                  </Button>
                  <Button size="sm" variant="ghost" disabled={del.isPending} onClick={() => setDeleting(s.name)}>
                    删除
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <ConfirmDialog
        open={deleting !== null}
        title="删除模板源"
        message={`确定删除「${deleting}」吗？cube create 将无法再按该名字引用，操作立即生效。`}
        confirmText="删除"
        danger
        onConfirm={() => {
          if (deleting) del.mutate({ name: deleting });
          setDeleting(null);
        }}
        onCancel={() => setDeleting(null)}
      />
      {editing && (
        <TplSourceForm
          key={editing === EMPTY_DRAFT ? 'new' : `edit:${editing.name}`}
          draft={editing}
          onClose={() => setEditing(null)}
        />
      )}
    </section>
  );
}
