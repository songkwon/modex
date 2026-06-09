"use client";

import { useEffect, useState } from "react";
import { Search, UserCircle } from "lucide-react";
import { CategoryTree } from "@/components/category-tree";
import { ModuleCard } from "@/components/module-card";
import { ModuleDrawer } from "@/components/module-drawer";
import { getAuthConfig, getCategories, getMe, getModules, logout, mockLogin } from "@/lib/api";
import type { AuthConfig, Category, ModuleInfo, User } from "@/types/modex";

export default function HomePage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selected, setSelected] = useState<ModuleInfo | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null);
  const [keyword, setKeyword] = useState("");

  useEffect(() => {
    getCategories().then(setCategories);
    getModules().then(setModules);
    getAuthConfig().then(setAuthConfig);
    getMe().then(setUser).catch(() => setUser(null));
  }, []);

  async function runSearch(value: string) {
    setKeyword(value);
    const q = value ? `?keyword=${encodeURIComponent(value)}` : "";
    setModules(await getModules(q));
  }

  async function login() {
    if (authConfig?.auth_mode === "oidc") {
      window.location.href = authConfig.login_url;
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
      <section className="grid home-grid">
        <CategoryTree categories={categories} />
        <div className="grid">
          <div className="panel">
            <div className="flex items-center gap-3">
              <Search size={18} />
              <input value={keyword} onChange={(e) => runSearch(e.target.value)} placeholder="搜索模块、关键词、维护人" />
            </div>
          </div>
          <div className="card-grid">
            {modules.map((module) => (
              <ModuleCard module={module} key={module.module_key} onInfo={setSelected} />
            ))}
          </div>
        </div>
        <aside className="grid content-start gap-4">
          <div className="panel">
            <div className="flex items-center gap-3">
              <UserCircle size={22} />
              <div>
                <div className="font-semibold">{user?.display_name || "未登录"}</div>
                <div className="muted text-sm">{user?.department || (authConfig?.auth_mode === "oidc" ? "Keycloak OIDC" : "Mock SSO")}</div>
              </div>
            </div>
            <div className="mt-4 flex flex-wrap gap-2">
              <button className="button" onClick={login}>{authConfig?.auth_mode === "oidc" ? "Keycloak 登录" : "Mock 登录"}</button>
              {user ? <button className="button" onClick={signOut}>退出</button> : null}
            </div>
          </div>
          <div className="panel">
            <h2 className="font-semibold">最近更新</h2>
            <div className="mt-3 grid gap-2 text-sm">
              {modules.slice(0, 3).map((m) => (
                <div key={m.module_key}>
                  <div>{m.name}</div>
                  <div className="muted">{m.updated_at.slice(0, 10)}</div>
                </div>
              ))}
            </div>
          </div>
          <div className="panel">
            <h2 className="font-semibold">热门文档</h2>
            <div className="mt-3 grid gap-2 text-sm">
              {modules.map((m) => (
                <div className="flex justify-between" key={m.module_key}>
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
