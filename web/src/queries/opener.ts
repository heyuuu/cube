// 打开意图（openerIntents 节，1038）：intent → 默认 opener + 候选清单。
// 候选缺省 = 声明了对应 role 的全部 opener（读侧合成，openers 数组可覆盖）。
export function useOpenerIntents() {
  return useQuery({
    queryKey: ['opener', 'intents'],
    queryFn: async () =>
      ((await apiGet('/api/opener/intents')).list ?? []).map((i) => ({ ...i, openers: i.openers ?? [] })),
  });
}

// opener 清单（openers 节）：project / workbench / md 等打开入口共用。
export function useOpenerList() {
  return useQuery({ queryKey: ['opener', 'list'], queryFn: () => apiGet('/api/opener/list') });
}

export function useIntentDefaultSave() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { intent: string; opener: string }) => apiPost('/api/opener/intent-default/save', input),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['opener', 'intents'] }),
  });
}

export function useIntentDefaultDelete() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { intent: string }) => apiPost('/api/opener/intent-default/delete', input),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['opener', 'intents'] }),
  });
}

// 解析某 intent 的默认 opener（组合 intents + openers 两个查询）。
// 返回取值函数：未配默认或 opener 失效（读侧已清理，这里兜底）时 undefined——
// 调用方隐藏入口，不回落其它 intent 的默认。
export function useIntentDefaultOpener() {
  const intents = useOpenerIntents();
  const openers = useOpenerList();
  return (intent: string): Opener | undefined => {
    const def = intents.data?.find((i) => i.intent === intent)?.defaultOpener;
    if (!def) {
      return undefined;
    }
    return (openers.data?.list ?? []).find((o) => o.name === def);
  };
}

// opener 增删改（openers 数据落 settings.json，经 Web API 写，保存即生效）。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiGet, apiPost } from '@/api/client';
import type { Opener, OpenerItem } from '@/api/client';

// 写后全量重拉：opener 清单变化会波及 intents（候选读侧合成、删除连带清引用），
// 刷整个 ['opener'] 前缀；返回 promise 让 mutation 在重拉完成后才 settle
export function useOpenerSave() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: OpenerItem) => apiPost('/api/opener/save', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['opener'] }),
  });
}

export function useOpenerDelete() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: OpenerItem) => apiPost('/api/opener/delete', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['opener'] }),
  });
}

// 整节覆盖（拖拽排序落库）：提交清单即落盘结果；失败也重拉，回落服务端顺序
export function useOpenerSaveAll() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (items: OpenerItem[]) => apiPost('/api/opener/save-all', { items }),
    onSettled: () => qc.invalidateQueries({ queryKey: ['opener'] }),
  });
}
