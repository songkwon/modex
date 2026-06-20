"use client";

import { ExternalLink, X } from "lucide-react";
import type { ModuleInfo } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

export function ModuleDrawer({ module, onClose }: { module: ModuleInfo | null; onClose: () => void }) {
  const { t } = useI18n();
  if (!module) return null;
  const rows = [
    [t("legacy.515559957fd3"), module.category_path],
    ["Owner", module.owner_group],
    ["maintainers", module.maintainers.join(", ")],
    [t("legacy.c68acbc684f6"), module.default_version],
    ["package_version", module.package_version],
    ["channel", module.channel],
    ["edition", module.edition],
    [t("legacy.c71895a3aa4b"), module.updated_at.slice(0, 10)]
  ];
  return (
    <aside className="drawer">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold">{module.name}</h2>
          <p className="muted mt-2 text-sm leading-6">{module.description}</p>
        </div>
        <button className="button icon-button" onClick={onClose} aria-label={t("legacy.3fd47edce45b")}>
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
        {t("legacy.921c3b62f1f7")}
      </a>
    </aside>
  );
}
