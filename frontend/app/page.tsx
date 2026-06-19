"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { FilePlus2, Layers3 } from "lucide-react";
import { ModuleDrawer } from "@/components/module-drawer";
import { DocSearch } from "@/components/doc-search";
import { PlatformCards } from "@/components/platform-cards";
import { EmptyState } from "@/components/ui/empty-state";
import { useSearch } from "@/components/search-provider";
import { getCategories, getModules } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import type { Category, ModuleInfo } from "@/types/modex";

export default function HomePage() {
  const { t } = useI18n();
  const router = useRouter();
  const { clearScope } = useSearch();
  const [categories, setCategories] = useState<Category[]>([]);
  const [allModules, setAllModules] = useState<ModuleInfo[]>([]);
  const [selected, setSelected] = useState<ModuleInfo | null>(null);
  const [aiActive, setAiActive] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      getCategories().then((items) => setCategories(Array.isArray(items) ? items : [])).catch(() => setCategories([])),
      getModules().then((items) => setAllModules(Array.isArray(items) ? items : [])).catch(() => setAllModules([])),
    ]).finally(() => setLoading(false));
    clearScope();
    const params = new URLSearchParams(window.location.search);
    const loginError = params.get("login_error");
    if (loginError) {
      alert(t("home.loginFailed", { error: loginError }));
      window.history.replaceState({}, "", window.location.pathname);
    }
    return () => clearScope();
  }, [clearScope, t]);

  function selectCategory(category: Category) {
    router.push(`/categories/${category.id}`);
  }

  const hasDocs = categories.length > 0 || allModules.length > 0;

  return (
    <main className="main">
      <section className={`home-hero mb-5 ${aiActive ? "home-hero-focus" : ""}`}>
        <div className="registry-kicker">
          <Layers3 size={16} />
          {t("home.kicker")}
        </div>
        <h1 className="registry-title">{t("home.title")}</h1>
        <p className="home-hero-sub muted">{t("home.subtitle")}</p>
        <DocSearch onAiActiveChange={setAiActive} />
      </section>

      {!aiActive ? (
        <section>
          <div className="shelf-toolbar mb-3">
            <div>
              <h2>{t("home.categoriesTitle")}</h2>
              <p>{t("home.categoriesDescription")}</p>
            </div>
          </div>
          {loading ? (
            <div className="empty-state">{t("home.loadingDocs")}</div>
          ) : hasDocs ? (
            <PlatformCards categories={categories} modules={allModules} onSelect={selectCategory} />
          ) : (
            <EmptyState
              icon={FilePlus2}
              title={t("home.emptyTitle")}
              hint={t("home.emptyHint")}
              action={
                <div className="empty-state-actions">
                  <Link className="button button-primary" href="/admin/modules">{t("home.emptyAction")}</Link>
                  <Link className="button" href="/me/guide">{t("home.emptyGuide")}</Link>
                </div>
              }
            />
          )}
        </section>
      ) : null}

      <ModuleDrawer module={selected} onClose={() => setSelected(null)} />
    </main>
  );
}
