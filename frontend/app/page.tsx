"use client";

import { useEffect, useState } from "react";
import { BookOpen, Clock3, Layers3, UserCircle } from "lucide-react";
import { CategoryTree } from "@/components/category-tree";
import { ModuleCard } from "@/components/module-card";
import { ModuleDrawer } from "@/components/module-drawer";
import { DocSearch } from "@/components/doc-search";
import { getAuthConfig, getCategories, getMe, getModules, logout, mockLogin } from "@/lib/api";
import type { AuthConfig, Category, ModuleInfo, User } from "@/types/modex";

export default function HomePage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selected, setSelected] = useState<ModuleInfo | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const frameworkCounts = modules.reduce<Record<string, number>>((acc, module) => {
    const framework = module.keywords.includes("fumadocs") ? "Fumadocs" : module.keywords.includes("vuepress") ? "VuePress" : "Markdown";
    acc[framework] = (acc[framework] || 0) + 1;
    return acc;
  }, {});
  const ownerCounts = modules.reduce<Record<string, number>>((acc, module) => {
    acc[module.owner_group] = (acc[module.owner_group] || 0) + 1;
    return acc;
  }, {});
  const popularModules = [...modules].sort((a, b) => b.reads_30d - a.reads_30d).slice(0, 5);
  const recentModules = [...modules].sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at)).slice(0, 5);

  useEffect(() => {
    getCategories().then(setCategories);
    getModules().then(setModules);
    getAuthConfig().then(setAuthConfig);
    getMe().then(setUser).catch(() => setUser(null));
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

  async function login() {
    if (authConfig?.auth_mode === "oidc") {
      if (authConfig.login_url) {
        window.location.href = authConfig.login_url;
      } else {
        alert("Keycloak OIDC 未完整配置：请检查 KEYCLOAK_BASE_URL / KEYCLOAK_REALM / OIDC_CLIENT_ID 等环境变量。");
      }
      return;
    }
    const res = await mockLogin();
    setUser(res.user);
  }

  async function signOut() {
    await logout();
    setUser(null);
  }

  return (
    <main className="main">
      <section className="home-hero mb-5">
        <div className="registry-kicker">
          <Layers3 size={16} />
          Modex Registry
        </div>
        <h1 className="registry-title">研发文档索引</h1>
        <p className="home-hero-sub muted">统一检索公司各平台研发文档，或直接向 AI 提问。</p>
        <DocSearch />
        <div className="home-hero-metrics">
          <span><strong>{modules.length}</strong> 文档集合</span>
          <span><strong>{Object.keys(frameworkCounts).length}</strong> 文档框架</span>
          <span><strong>{Object.keys(ownerCounts).length}</strong> Owner 团队</span>
        </div>
      </section>
      <section className="grid home-grid">
        <CategoryTree activeID={selectedCategory?.id} categories={categories} onSelect={selectCategory} />
        <div className="grid">
          <div className="shelf-toolbar">
            <div>
              <h2>{selectedCategory ? selectedCategory.name : "全部文档"}</h2>
              <p>{modules.length} 个文档集合，按最近发布和阅读热度混排。</p>
            </div>
            <div className="shelf-tabs compact">
              {Object.entries(frameworkCounts).map(([name, count]) => (
                <span className="shelf-tab" key={name}>{name} {count}</span>
              ))}
            </div>
          </div>
          <div className="package-list">
            {modules.map((module) => (
              <ModuleCard module={module} key={module.module_key} onInfo={setSelected} />
            ))}
            {modules.length === 0 ? <div className="empty-state">没有匹配的文档集合</div> : null}
          </div>
        </div>
        <aside className="grid content-start gap-4 rail">
          <div className="identity-panel">
            <div className="flex items-center gap-3">
              <UserCircle size={22} />
              <div>
                <div className="font-semibold">{user?.display_name || "未登录"}</div>
                <div className="muted text-sm">{user?.department || (authConfig?.auth_mode === "oidc" ? "Keycloak OIDC" : "Mock SSO")}</div>
              </div>
            </div>
            <div className="mt-4 flex flex-wrap gap-2">
              <button className="button button-primary" onClick={login}>{authConfig?.auth_mode === "oidc" ? "Keycloak 登录" : "Mock 登录"}</button>
              {user ? <button className="button" onClick={signOut}>退出</button> : null}
            </div>
          </div>
          <div className="panel">
            <div className="section-heading">
              <h2>文档类型</h2>
              <BookOpen size={16} className="text-blue-600" />
            </div>
            <div className="facet-stack">
              {Object.entries(frameworkCounts).map(([name, count]) => (
                <div className="facet-row" key={name}>
                  <span>{name}</span>
                  <strong>{count}</strong>
                </div>
              ))}
            </div>
          </div>
          <div className="panel">
            <div className="section-heading">
              <h2>最近更新</h2>
              <Clock3 size={16} className="text-blue-600" />
            </div>
            <div className="text-sm">
              {recentModules.map((m) => (
                <div className="activity-row" key={m.module_key}>
                  <div>{m.name}</div>
                  <div className="muted">{m.updated_at.slice(0, 10)}</div>
                </div>
              ))}
            </div>
          </div>
          <div className="panel">
            <div className="section-heading">
              <h2>按 Owner</h2>
              <span className="tag">team</span>
            </div>
            <div className="text-sm">
              {Object.entries(ownerCounts).map(([name, count]) => (
                <div className="activity-row" key={name}>
                  <span>{name}</span>
                  <span className="muted">{count}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="panel">
            <div className="section-heading">
              <h2>热门文档</h2>
              <span className="tag">30d</span>
            </div>
            <div className="text-sm">
              {popularModules.map((m) => (
                <div className="activity-row" key={m.module_key}>
                  <span>{m.name}</span>
                  <span className="muted">{m.reads_30d}</span>
                </div>
              ))}
            </div>
          </div>
        </aside>
      </section>
      <ModuleDrawer module={selected} onClose={() => setSelected(null)} />
    </main>
  );
}
