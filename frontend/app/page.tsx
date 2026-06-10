"use client";

import { useEffect, useState } from "react";
import { ArrowLeft, Layers3 } from "lucide-react";
import { CategoryTree } from "@/components/category-tree";
import { ModuleCard } from "@/components/module-card";
import { ModuleDrawer } from "@/components/module-drawer";
import { DocSearch } from "@/components/doc-search";
import { PlatformCards } from "@/components/platform-cards";
import { useSearch } from "@/components/search-provider";
import { getCategories, getModules } from "@/lib/api";
import type { Category, ModuleInfo } from "@/types/modex";

export default function HomePage() {
  const { setScope, clearScope } = useSearch();
  const [categories, setCategories] = useState<Category[]>([]);
  const [allModules, setAllModules] = useState<ModuleInfo[]>([]);
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selected, setSelected] = useState<ModuleInfo | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const [aiActive, setAiActive] = useState(false);

  useEffect(() => {
    getCategories().then(setCategories);
    getModules().then(setAllModules);
    clearScope();
    const params = new URLSearchParams(window.location.search);
    const loginError = params.get("login_error");
    if (loginError) {
      alert("登录失败：" + loginError);
      window.history.replaceState({}, "", window.location.pathname);
    }
    return () => clearScope();
  }, [clearScope]);

  async function selectCategory(category: Category | null) {
    setSelectedCategory(category);
    if (!category) {
      clearScope();
      return;
    }
    // Scope the top-bar search (and ⌘K palette) to this capability domain.
    setScope({ categoryId: category.id, label: category.name });
    const query = `?category_id=${encodeURIComponent(category.id)}`;
    setModules(await getModules(query));
  }

  return (
    <main className="main">
      {/* The big hero search only appears on the landing view, not inside a domain. */}
      {!selectedCategory ? (
        <section className={`home-hero mb-5 ${aiActive ? "home-hero-focus" : ""}`}>
          <div className="registry-kicker">
            <Layers3 size={16} />
            Modex Registry
          </div>
          <h1 className="registry-title">研发文档索引</h1>
          <p className="home-hero-sub muted">统一检索公司各平台研发文档，或直接向 AI 提问。</p>
          <DocSearch onAiActiveChange={setAiActive} />
        </section>
      ) : null}

      {/* Capability domains — hidden while focusing on the AI conversation. */}
      {!aiActive && !selectedCategory ? (
        <section>
          <div className="shelf-toolbar mb-3">
            <div>
              <h2>能力域</h2>
              <p>按公司平台划分的文档集合，点击平台或子平台进入。</p>
            </div>
          </div>
          <PlatformCards categories={categories} modules={allModules} onSelect={selectCategory} />
        </section>
      ) : null}

      {selectedCategory ? (
        <section className="grid home-grid">
          <CategoryTree activeID={selectedCategory.id} categories={categories} onSelect={selectCategory} />
          <div className="grid">
            <div className="shelf-toolbar">
              <div>
                <button className="button mb-2" onClick={() => selectCategory(null)}><ArrowLeft size={15} />全部能力域</button>
                <h2>{selectedCategory.name}</h2>
                <p>{modules.length} 个文档集合 · 顶栏搜索（⌘K）已限定在「{selectedCategory.name}」</p>
              </div>
            </div>
            <div className="package-list">
              {modules.map((module) => (
                <ModuleCard module={module} key={module.module_key} onInfo={setSelected} />
              ))}
              {modules.length === 0 ? <div className="empty-state">该能力域下暂无文档集合</div> : null}
            </div>
          </div>
        </section>
      ) : null}

      <ModuleDrawer module={selected} onClose={() => setSelected(null)} />
    </main>
  );
}
