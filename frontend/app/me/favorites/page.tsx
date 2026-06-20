"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowUpRight, Heart, Trash2 } from "lucide-react";
import { EmptyState } from "@/components/ui/empty-state";
import { favoriteModuleKeys, favoriteModulesFrom, syncSetFavoriteModule, syncedFavoriteModuleKeys } from "@/lib/local-docs";
import { getModules } from "@/lib/api";
import type { ModuleInfo } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

export default function FavoritesPage() {
  const { t } = useI18n();
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [favoriteKeys, setFavoriteKeys] = useState<string[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    setFavoriteKeys(favoriteModuleKeys());
    syncedFavoriteModuleKeys().then(setFavoriteKeys);
    getModules()
      .then(setModules)
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
    const refresh = () => setFavoriteKeys(favoriteModuleKeys());
    window.addEventListener("modex:favorites-changed", refresh);
    return () => window.removeEventListener("modex:favorites-changed", refresh);
  }, []);

  const favorites = favoriteModulesFrom(modules);

  function remove(moduleKey: string) {
    syncSetFavoriteModule(moduleKey, false).then(setFavoriteKeys);
  }

  return (
    <main className="main">
      <section className="hero-panel compact-hero">
        <span className="hero-eyebrow">Personal</span>
        <h1 className="hero-title">{t("me.favorites.my_follows")}</h1>
        <p className="hero-copy">{t("me.favorites.frequently_viewed_document_sources_are_grouped_here")}</p>
      </section>

      {error ? <div className="panel badge-danger mt-5">{t("admin.mcpLogs.load_failed")}{error}</div> : null}

      <section className="mt-5">
        {favorites.length === 0 ? (
          <div className="table-card">
            <EmptyState
              icon={Heart}
              title={favoriteKeys.length ? t("me.favorites.followed_modules_are_temporarily_unavailable") : t("me.favorites.no_modules_followed_yet")}
              hint={favoriteKeys.length ? t("me.favorites.these_modules_may_have_been_deprecated_or_renamed") : t("me.favorites.followed_modules_will_appear_here")}
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
                    <dt className="muted">{t("categories.id.default_version")}</dt>
                    <dd>{module.default_version}</dd>
                  </div>
                  <div>
                    <dt className="muted">{t("categories.id.reads_30d")}</dt>
                    <dd>{module.reads_30d}</dd>
                  </div>
                  <div className="flex gap-2 pt-1">
                    <button className="button icon-button" aria-label={t("me.favorites.unfollow")} onClick={() => remove(module.module_key)}>
                      <Trash2 size={15} />
                    </button>
                    <Link className="button" href={`/docs/${module.module_key}/${module.default_version}`}>{t("categories.id.open")} <ArrowUpRight size={14} /></Link>
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
