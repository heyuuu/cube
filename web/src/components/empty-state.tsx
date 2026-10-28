// 空状态：标题 + 副文案
export function EmptyState({ title, sub }: { title: string; sub?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="text-sm font-medium">{title}</div>
      {sub ? <div className="text-muted-foreground mt-1 text-xs">{sub}</div> : null}
    </div>
  );
}
