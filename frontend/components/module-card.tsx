"use client";

import { ArrowUpRight, BookOpen, Heart, Info } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { isFavoriteModule, setFavoriteModule } from "@/lib/local-docs";
import type { ModuleInfo } from "@/types/modex";

export function ModuleCard({ module, onInfo }: { module: ModuleInfo; onInfo: (module: ModuleInfo) => void }) {
  const [favorite, setFavorite] = useState(false);

  useEffect(() => {
    setFavorite(isFavoriteModule(module.module_key));
  }, [module.module_key]);

  function toggleFavorite() {
    setFavorite((current) => {
      const next = !current;
      setFavoriteModule(module.module_key, next);
      return next;
    });
  }

  return (
    <article className="package-row">
      <div>
        <Link href={`/docs/${module.module_key}/${module.default_version}`} className="min-w-0">
          <div className="package-name">
            <span className="status-dot" />
            <span>{module.name}</span>
          </div>
        </Link>
        <p className="muted mt-3 text-sm leading-6">{module.description}</p>
        <div className="package-meta">
          <span>{module.category_path}</span>
          <span>owner: {module.owner_group}</span>
          <span>channel: {module.channel}</span>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          {(module.keywords || []).map((tag) => (
            <span className="tag" key={tag}>{tag}</span>
          ))}
        </div>
      </div>
      <aside className="package-stats text-sm">
        <span className="score-pill">active package</span>
        <div>
          <dt className="muted">默认版本</dt>
          <dd>{module.default_version}</dd>
        </div>
        <div>
          <dt className="muted">文档类型</dt>
          <dd className="flex items-center gap-1"><BookOpen size={14} /> {(module.keywords || []).includes("fumadocs") ? "Fumadocs" : (module.keywords || []).includes("vuepress") ? "VuePress" : "Markdown"}</dd>
        </div>
        <div>
          <dt className="muted">阅读 30d</dt>
          <dd>{module.reads_30d}</dd>
        </div>
        <div>
          <dt className="muted">最近更新</dt>
          <dd>{module.updated_at.slice(0, 10)}</dd>
        </div>
        <div className="flex gap-2 pt-1">
          <button className={`button icon-button${favorite ? " is-active" : ""}`} aria-label={favorite ? "取消关注" : "关注模块"} onClick={toggleFavorite}>
            <Heart size={15} fill={favorite ? "currentColor" : "none"} />
          </button>
          <button className="button icon-button" aria-label="模块信息" onClick={() => onInfo(module)}><Info size={15} /></button>
          <Link className="button" href={`/docs/${module.module_key}/${module.default_version}`}>打开 <ArrowUpRight size={14} /></Link>
        </div>
      </aside>
    </article>
  );
}
