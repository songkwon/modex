"use client";

import { ExternalLink, X } from "lucide-react";
import type { ModuleInfo } from "@/types/modex";

export function ModuleDrawer({ module, onClose }: { module: ModuleInfo | null; onClose: () => void }) {
  if (!module) return null;
  const rows = [
    ["分类", module.category_path],
    ["Owner", module.owner_group],
    ["maintainers", module.maintainers.join(", ")],
    ["默认文档版本", module.default_version],
    ["package_version", module.package_version],
    ["channel", module.channel],
    ["edition", module.edition],
    ["最近发布", module.updated_at.slice(0, 10)]
  ];
  return (
    <aside className="drawer">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold">{module.name}</h2>
          <p className="muted mt-2 text-sm leading-6">{module.description}</p>
        </div>
        <button className="button icon-button" onClick={onClose} aria-label="关闭">
          <X size={16} />
        </button>
      </div>
      <dl className="mt-5 grid gap-2 rounded-2xl border border-border bg-slate-50/70 p-4">
        {rows.map(([label, value]) => (
          <div className="grid grid-cols-[120px_1fr] gap-3 border-t border-border py-2 text-sm first:border-t-0 first:pt-0 last:pb-0" key={label}>
            <dt className="muted">{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      <div className="mt-5 flex flex-wrap gap-2">
        {module.keywords.map((tag) => (
          <span className="tag" key={tag}>{tag}</span>
        ))}
      </div>
      <a className="button mt-5" href={module.repo_url}>
        <ExternalLink size={16} />
        查看源码
      </a>
    </aside>
  );
}
