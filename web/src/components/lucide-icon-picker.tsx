// lucide 图标选择器：当前值预览 + 搜索过滤 + 网格点选（icon 编辑表单用，见 IconField）。
// icons 映射键是 PascalCase，存储值是 kebab-case，此处预计算双名做过滤与高亮。
import { icons } from 'lucide-react';
import { createElement, useMemo, useState } from 'react';
import type { LucideIcon } from 'lucide-react';

import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

function pascalToKebab(name: string): string {
  return name.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase();
}

type Entry = { kebab: string; Ico: LucideIcon };
const ALL: Entry[] = Object.entries(icons).map(([pascal, Ico]) => ({ kebab: pascalToKebab(pascal), Ico }));
const BY_KEBAB = new Map(ALL.map((e) => [e.kebab, e.Ico]));
const MAX_SHOWN = 96;

export function LucideIconPicker({ value, onChange }: { value: string; onChange: (kebab: string) => void }) {
  const [query, setQuery] = useState('');
  const hits = useMemo(() => {
    const q = query.trim().toLowerCase();
    const list = q ? ALL.filter((e) => e.kebab.includes(q)) : ALL;
    return list.slice(0, MAX_SHOWN);
  }, [query]);
  // 预览按存储值直接取图；未知图名（如手工改过 settings.json）退化为纯文本展示
  const preview = BY_KEBAB.get(value);

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center gap-2">
        {preview && createElement(preview, { className: 'size-8' })}
        <span className="text-muted-foreground text-xs">{value || '未选择'}</span>
      </div>
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="搜索图标（如 folder / git / terminal）"
      />
      <div className="grid max-h-52 grid-cols-8 gap-1 overflow-y-auto rounded-md border p-2">
        {hits.map(({ kebab, Ico }) => (
          <button
            key={kebab}
            type="button"
            title={kebab}
            aria-label={kebab}
            aria-pressed={kebab === value}
            onClick={() => onChange(kebab)}
            className={cn(
              'flex size-8 items-center justify-center rounded-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground',
              kebab === value && 'bg-primary/15 text-primary hover:bg-primary/15 hover:text-primary',
            )}
          >
            <Ico className="size-4" />
          </button>
        ))}
        {hits.length === 0 && (
          <div className="text-muted-foreground col-span-8 py-4 text-center text-xs">无匹配图标</div>
        )}
      </div>
      {/* 命中裁剪提示：选中项可能不在可视区，当前值以上方预览为准 */}
      {ALL.length > MAX_SHOWN && hits.length === MAX_SHOWN && (
        <div className="text-muted-foreground text-[10px]">
          显示前 {MAX_SHOWN} / {ALL.length} 个，搜索缩小范围
        </div>
      )}
    </div>
  );
}
