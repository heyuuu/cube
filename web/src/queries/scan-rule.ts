// scan/clone 规则增删改（数据落 settings.json，经 Web API 写，保存即生效）。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';

import { apiGet, apiPost } from '@/api/client';
import type { CloneRuleItem, ScanRuleItem } from '@/api/client';

const RULE_KEYS = ['project', 'scan-rules'] as const;
const CLONE_KEYS = ['project', 'clone-rules'] as const;

export function useScanRules() {
  return useQuery({ queryKey: [...RULE_KEYS], queryFn: () => apiGet('/api/project/scan-rules') });
}

export function useCloneRules() {
  return useQuery({ queryKey: [...CLONE_KEYS], queryFn: () => apiGet('/api/project/clone-rules') });
}

// 写后全量重拉：scan 规则写成功后端即重扫，项目列表一并刷新；返回 promise 让 mutation
// 在重拉完成后才 settle
function refreshScan(qc: QueryClient) {
  return Promise.all([
    qc.invalidateQueries({ queryKey: [...RULE_KEYS] }),
    qc.invalidateQueries({ queryKey: ['project', 'list'] }),
  ]);
}

export function useScanRuleSave() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ScanRuleItem) => apiPost('/api/project/scan-rule/save', input),
    onSuccess: () => refreshScan(qc),
  });
}

export function useScanRuleDelete() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ScanRuleItem) => apiPost('/api/project/scan-rule/delete', input),
    onSuccess: () => refreshScan(qc),
  });
}

// 整节覆盖（拖拽排序落库）：提交清单即落盘结果；失败也重拉，回落服务端顺序
export function useScanRuleSaveAll() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (items: ScanRuleItem[]) => apiPost('/api/project/scan-rule/save-all', { items }),
    onSettled: () => refreshScan(qc),
  });
}

export function useCloneRuleSave() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CloneRuleItem) => apiPost('/api/project/clone-rule/save', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [...CLONE_KEYS] }),
  });
}

export function useCloneRuleDelete() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CloneRuleItem) => apiPost('/api/project/clone-rule/delete', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [...CLONE_KEYS] }),
  });
}

export function useCloneRuleSaveAll() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (items: CloneRuleItem[]) => apiPost('/api/project/clone-rule/save-all', { items }),
    onSettled: () => qc.invalidateQueries({ queryKey: [...CLONE_KEYS] }),
  });
}
