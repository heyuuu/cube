// 模板源配置（settings.json tplSources 节，经 Web API 写，保存即生效）。
// cube create 按 name 引用模板源（@name/模板名），缺省源为 core。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiGet, apiPost } from '@/api/client';
import type { TplSourceItem } from '@/api/client';

const TPL_KEYS = ['template'] as const;

export function useTplSources() {
  return useQuery({ queryKey: [...TPL_KEYS, 'sources'], queryFn: () => apiGet('/api/template/source/list') });
}

// 写后全量重拉；返回 promise 让 mutation 在重拉完成后才 settle
export function useTplSourceSave() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: TplSourceItem) => apiPost('/api/template/source/save', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [...TPL_KEYS] }),
  });
}

export function useTplSourceDelete() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: TplSourceItem) => apiPost('/api/template/source/delete', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [...TPL_KEYS] }),
  });
}

// 整节覆盖（拖拽排序落库）：提交清单即落盘结果；失败也重拉，回落服务端顺序
export function useTplSourceSaveAll() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (items: TplSourceItem[]) => apiPost('/api/template/source/save-all', { items }),
    onSettled: () => qc.invalidateQueries({ queryKey: [...TPL_KEYS] }),
  });
}
