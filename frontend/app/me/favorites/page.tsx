"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowUpRight, Heart, Trash2 } from "lucide-react";
import { EmptyState } from "@/components/ui/empty-state";
import { favoriteModuleKeys, favoriteModulesFrom, setFavoriteModule } from "@/lib/local-docs";
import { getModules } from "@/lib/api";
import type { ModuleInfo } from "@/types/modex";

export default function FavoritesPage() {
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [favoriteKeys, setFavoriteKeys] = useState<string[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    setFavoriteKeys(favoriteModuleKeys());
    getModules()
      .then(setModules)
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
    const refresh = () => setFavoriteKeys(favoriteModuleKeys());
    window.addEventListener("modex:favorites-changed", refresh);
    return () => window.removeEventListener("modex:favorites-changed", refresh);
  }, []);

  const favorites = favoriteModulesFrom(modules);

  function remove(moduleKey: string) {
    setFavoriteKeys(setFavoriteModule(moduleKey, false));
  }

  return (
    <main className="main">
      <section className="hero-panel compact-hero">
        <span className="hero-eyebrow">Personal</span>
        <h1 className="hero-title">我的关注</h1>
        <p className="hero-copy">常看的文档源集中在这里。</p>
      </section>

      {error ? <div className="panel badge-danger mt-5">加载失败：{error}</div> : null}

      <section className="mt-5">
        {favorites.length === 0 ? (
          <div className="table-card">
            <EmptyState
              icon={Heart}
              title={favoriteKeys.length ? "关注的模块暂时不可用" : "还没有关注模块"}
              hint={favoriteKeys.length ? "这些模块可能已被下线或改名。" : "关注后的模块会出现在这里。"}
            />
          </div>
        ) : (
          <div className="module-list">
            {favorites.map((module) => (
              <article className="package-row" key={module.module_key}>
                <div>
                  <Link href={`/docs/${module.module_key}/${module.default_version}`} className="package-name">
                    <span className="status-dot" />
                    <span>{module.name}</span>
                  </Link>
                  <p className="muted mt-3 text-sm leading-6">{module.description}</p>
                  <div className="package-meta">
                    <span>{module.category_path}</span>
                    <span>owner: {module.owner_group}</span>
                    <span>updated: {module.updated_at?.slice(0, 10)}</span>
                  </div>
                </div>
                <aside className="package-stats text-sm">
                  <span className="score-pill">favorite</span>
                  <div>
                    <dt className="muted">默认版本</dt>
                    <dd>{module.default_version}</dd>
                  </div>
                  <div>
                    <dt className="muted">阅读 30d</dt>
                    <dd>{module.reads_30d}</dd>
                  </div>
                  <div className="flex gap-2 pt-1">
                    <button className="button icon-button" aria-label="取消关注" onClick={() => remove(module.module_key)}>
                      <Trash2 size={15} />
                    </button>
                    <Link className="button" href={`/docs/${module.module_key}/${module.default_version}`}>打开 <ArrowUpRight size={14} /></Link>
                  </div>
                </aside>
              </article>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
