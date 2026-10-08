// 模板源配置（settings.json tplSources 节，经 Web API 写，保存即生效）。
// cube create 按 name 引用模板源（@name/模板名），缺省源为 core。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiGet, apiPost } from '@/api/client';

const TPL_KEYS = ['template'] as const;

export function useTplSources() {
  return useQuery({ queryKey: [...TPL_KEYS, 'sources'], queryFn: () => apiGet('/api/template/source/list') });
}

export function useTplSourceSave() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: Parameters<typeof apiPost<'/api/template/source/save'>>[1]) =>
      apiPost('/api/template/source/save', input),
    onSuccess: () => void qc.invalidateQueries({ queryKey: [...TPL_KEYS] }),
  });
}

export function useTplSourceDelete() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { name: string }) => apiPost('/api/template/source/delete', input),
    onSuccess: () => void qc.invalidateQueries({ queryKey: [...TPL_KEYS] }),
  });
}

// 拖拽排序：提交按目标顺序排列的全量名单（乐观更新在调用方，失败时 invalidate 回滚）
export function useTplSourceReorder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: Parameters<typeof apiPost<'/api/template/source/reorder'>>[1]) =>
      apiPost('/api/template/source/reorder', input),
    onSuccess: () => void qc.invalidateQueries({ queryKey: [...TPL_KEYS] }),
    onError: () => void qc.invalidateQueries({ queryKey: [...TPL_KEYS] }),
  });
}
