// 内容区占位：代码阅读（1012）/ diff（1013）按选中态在此切换
export function ContentPanelPlaceholder({ title }: { title: React.ReactNode }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 text-center">
      <div className="text-muted-foreground text-sm font-medium">{title}</div>
      <div className="text-muted-foreground/70 text-xs">代码阅读（1012）与 diff（1013）将在此呈现</div>
    </div>
  );
}
