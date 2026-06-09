"use client";

import { Info, Star } from "lucide-react";
import Link from "next/link";
import type { ModuleInfo } from "@/types/modex";

export function ModuleCard({ module, onInfo }: { module: ModuleInfo; onInfo: (module: ModuleInfo) => void }) {
  return (
    <article className="card">
      <div className="flex items-start justify-between gap-3">
        <Link href={`/docs/${module.module_key}/${module.default_version}`} className="min-w-0">
          <h3 className="text-lg font-semibold">{module.name}</h3>
          <p className="muted mt-1 text-sm">{module.category_path}</p>
        </Link>
        <div className="flex gap-2">
          <button className="button" aria-label="关注">
            <Star size={16} />
          </button>
          <button className="button" aria-label="模块信息" onClick={() => onInfo(module)}>
            <Info size={16} />
          </button>
        </div>
      </div>
      <p className="mt-3 text-sm leading-6">{module.description}</p>
      <dl className="mt-4 grid grid-cols-2 gap-3 text-sm">
        <div>
          <dt className="muted">默认版本</dt>
          <dd>{module.default_version}</dd>
        </div>
        <div>
          <dt className="muted">工程版本</dt>
          <dd>{module.package_version}</dd>
        </div>
        <div>
          <dt className="muted">状态</dt>
          <dd>{module.status}</dd>
        </div>
        <div>
          <dt className="muted">最近更新</dt>
          <dd>{module.updated_at.slice(0, 10)}</dd>
        </div>
      </dl>
      <div className="mt-4 flex flex-wrap gap-2">
        {module.keywords.map((tag) => (
          <span className="tag" key={tag}>{tag}</span>
        ))}
      </div>
    </article>
  );
}
