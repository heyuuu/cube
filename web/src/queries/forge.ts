// forge 配置与拉取（数据落 settings.json forges / forgeAccounts 节，经 Web API 写，保存即生效）。
// 1044 起 namespace 模型移除，account 是 forge 下唯一的关联配置。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiGet, apiPost } from '@/api/client';
import type { ForgeAccountItem, ForgeItem } from '@/api/client';

const FORGE_KEYS = ['forge'] as const;

export function useForges() {
  return useQuery({ queryKey: [...FORGE_KEYS], queryFn: () => apiGet('/api/forge/list') });
}

// 写后全量重拉：['forge'] 前缀覆盖 list / accounts / overview（删 forge 级联删 account）；
// 返回 promise 让 mutation 在重拉完成后才 settle
export function useForgeSave() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ForgeItem) => apiPost('/api/forge/save', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [...FORGE_KEYS] }),
  });
}

export function useForgeDelete() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ForgeItem) => apiPost('/api/forge/delete', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [...FORGE_KEYS] }),
  });
}

// 整节覆盖（拖拽排序落库）：提交清单即落盘结果；失败也重拉，回落服务端顺序
export function useForgeSaveAll() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (items: ForgeItem[]) => apiPost('/api/forge/save-all', { items }),
    onSettled: () => qc.invalidateQueries({ queryKey: [...FORGE_KEYS] }),
  });
}

// --- account（token 掩码语义：list 打码返回，save 提交掩码值 = 未修改） ---

export function useForgeAccounts() {
  return useQuery({ queryKey: [...FORGE_KEYS, 'accounts'], queryFn: () => apiGet('/api/forge/account/list') });
}

export function useForgeAccountSave() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ForgeAccountItem) => apiPost('/api/forge/account/save', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [...FORGE_KEYS] }),
  });
}

// 提交 list 回显行即可（token 为掩码值，按唯一键定位，不参与匹配）
export function useForgeAccountDelete() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ForgeAccountItem) => apiPost('/api/forge/account/delete', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [...FORGE_KEYS] }),
  });
}

// 整节覆盖（拖拽排序落库）：list 回显的掩码 token 原样提交，后端按唯一键沿用旧值
export function useForgeAccountSaveAll() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (items: ForgeAccountItem[]) => apiPost('/api/forge/account/save-all', { items }),
    onSettled: () => qc.invalidateQueries({ queryKey: [...FORGE_KEYS] }),
  });
}

// 拉取 account 名下全部仓库：出站 API 调用（慢操作），成功返回 count
export function useForgeAccountFetch() {
  return useMutation({
    mutationFn: (input: { forgeHost: string; username: string; force?: boolean }) =>
      apiPost('/api/forge/account/fetch', input),
  });
}

// forge 页聚合（1042/1044）：全部 account 对账行 + 拉取元信息（只读缓存，不外呼）
export function useForgeOverview() {
  return useQuery({ queryKey: [...FORGE_KEYS, 'overview'], queryFn: () => apiGet('/api/forge/overview') });
}
