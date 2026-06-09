"use client";

import { useEffect, useState } from "react";
import { Layers3 } from "lucide-react";
import { CategoryTree } from "@/components/category-tree";
import { ModuleCard } from "@/components/module-card";
import { ModuleDrawer } from "@/components/module-drawer";
import { DocSearch } from "@/components/doc-search";
import { getCategories, getModules } from "@/lib/api";
import type { Category, ModuleInfo } from "@/types/modex";

export default function HomePage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selected, setSelected] = useState<ModuleInfo | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const [aiActive, setAiActive] = useState(false);

  useEffect(() => {
    getCategories().then(setCategories);
    getModules().then(setModules);
    const params = new URLSearchParams(window.location.search);
    const loginError = params.get("login_error");
    if (loginError) {
      alert("登录失败：" + loginError);
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, []);

  async function selectCategory(category: Category | null) {
    setSelectedCategory(category);
    const query = category ? `?category_id=${encodeURIComponent(category.id)}` : "";
    setModules(await getModules(query));
  }

  return (
    <main className="main">
      <section className={`home-hero mb-5 ${aiActive ? "home-hero-focus" : ""}`}>
        <div className="registry-kicker">
          <Layers3 size={16} />
          Modex Registry
        </div>
        <h1 className="registry-title">研发文档索引</h1>
        <p className="home-hero-sub muted">统一检索公司各平台研发文档，或直接向 AI 提问。</p>
        <DocSearch onAiActiveChange={setAiActive} />
      </section>

      {/* Capability domains — hidden while focusing on the AI conversation. */}
      {!aiActive ? (
        <section className="grid home-grid">
          <CategoryTree activeID={selectedCategory?.id} categories={categories} onSelect={selectCategory} />
          <div className="grid">
            <div className="shelf-toolbar">
              <div>
                <h2>{selectedCategory ? selectedCategory.name : "全部能力域"}</h2>
                <p>{modules.length} 个文档集合{selectedCategory ? ` · ${selectedCategory.name}` : ""}</p>
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
