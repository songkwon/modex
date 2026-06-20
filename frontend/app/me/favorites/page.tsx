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
        <h1 className="hero-title">{t("legacy.9dccba0eafbb")}</h1>
        <p className="hero-copy">{t("legacy.612f57738c06")}</p>
      </section>

      {error ? <div className="panel badge-danger mt-5">{t("legacy.01de8216e0d0")}{error}</div> : null}

      <section className="mt-5">
        {favorites.length === 0 ? (
          <div className="table-card">
            <EmptyState
              icon={Heart}
              title={favoriteKeys.length ? t("legacy.34cacdaed351") : t("legacy.9d64d2c02081")}
              hint={favoriteKeys.length ? t("legacy.76b0c12684e6") : t("legacy.379856f658f6")}
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
                    <dt className="muted">{t("legacy.a83a27e9be90")}</dt>
                    <dd>{module.default_version}</dd>
                  </div>
                  <div>
                    <dt className="muted">{t("legacy.265cd352bea6")}</dt>
                    <dd>{module.reads_30d}</dd>
                  </div>
                  <div className="flex gap-2 pt-1">
                    <button className="button icon-button" aria-label={t("legacy.dc89d336c8e9")} onClick={() => remove(module.module_key)}>
                      <Trash2 size={15} />
                    </button>
                    <Link className="button" href={`/docs/${module.module_key}/${module.default_version}`}>{t("legacy.c771248e511f")} <ArrowUpRight size={14} /></Link>
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
