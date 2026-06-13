"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Layers3 } from "lucide-react";
import { ModuleDrawer } from "@/components/module-drawer";
import { DocSearch } from "@/components/doc-search";
import { PlatformCards } from "@/components/platform-cards";
import { useSearch } from "@/components/search-provider";
import { getCategories, getModules } from "@/lib/api";
import type { Category, ModuleInfo } from "@/types/modex";

export default function HomePage() {
  const router = useRouter();
  const { clearScope } = useSearch();
  const [categories, setCategories] = useState<Category[]>([]);
  const [allModules, setAllModules] = useState<ModuleInfo[]>([]);
  const [selected, setSelected] = useState<ModuleInfo | null>(null);
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

  function selectCategory(category: Category) {
    router.push(`/categories/${category.id}`);
  }

  return (
    <main className="main">
      <section className={`home-hero mb-5 ${aiActive ? "home-hero-focus" : ""}`}>
        <div className="registry-kicker">
          <Layers3 size={16} />
          Modex Registry
        </div>
        <h1 className="registry-title">文档中心</h1>
        <p className="home-hero-sub muted">统一检索所有项目文档，或直接向 AI 提问。</p>
        <DocSearch onAiActiveChange={setAiActive} />
      </section>

      {!aiActive ? (
        <section>
          <div className="shelf-toolbar mb-3">
            <div>
              <h2>分类</h2>
              <p>按分类组织的文档集合，点击分类或子分类进入。</p>
            </div>
          </div>
          <PlatformCards categories={categories} modules={allModules} onSelect={selectCategory} />
        </section>
      ) : null}

      <ModuleDrawer module={selected} onClose={() => setSelected(null)} />
    </main>
  );
}
