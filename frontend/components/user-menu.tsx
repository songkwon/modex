"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Clock3, Star, Terminal, Shield, LogOut, LogIn, ChevronDown } from "lucide-react";
import { getAuthConfig, getMe, logout, mockLogin } from "@/lib/api";
import type { AuthConfig, User } from "@/types/modex";

function initials(name: string) {
  const n = (name || "").trim();
  return n ? n.slice(0, 1).toUpperCase() : "?";
}

function UserAvatar({ user }: { user: { display_name?: string; username?: string; avatar?: string } }) {
  const name = user.display_name || user.username || "";
  if (user.avatar) {
    return (
      <img
        src={user.avatar}
        alt={name}
        className="user-avatar"
        style={{ objectFit: "cover" }}
        onError={(e) => {
          // fallback to initials on broken image
          const span = document.createElement("span");
          span.className = "user-avatar";
          span.textContent = initials(name);
          e.currentTarget.replaceWith(span);
        }}
      />
    );
  }
  return <span className="user-avatar">{initials(name)}</span>;
}

export function UserMenu() {
  const [user, setUser] = useState<User | null>(null);
  const [cfg, setCfg] = useState<AuthConfig | null>(null);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    getMe().then(setUser).catch(() => setUser(null));
    getAuthConfig().then(setCfg).catch(() => {});
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  async function login() {
    if (cfg?.auth_mode === "oidc") {
      if (cfg.login_url) window.location.href = cfg.login_url;
      else alert("Keycloak OIDC 未完整配置：请检查 KEYCLOAK_BASE_URL / KEYCLOAK_REALM / OIDC_CLIENT_ID 等环境变量。");
      return;
    }
    const res = await mockLogin();
    setUser(res.user);
  }

  async function signOut() {
    await logout();
    setUser(null);
    setOpen(false);
  }

  if (!user) {
    return (
      <button className="button button-primary" onClick={login}>
        <LogIn size={16} />登录
      </button>
    );
  }

  const isAdmin = (user.roles || []).includes("admin") || user.is_super_admin;

  return (
    <div className="user-menu" ref={ref} onMouseEnter={() => setOpen(true)} onMouseLeave={() => setOpen(false)}>
      <button className="user-menu-trigger" onClick={() => setOpen((v) => !v)}>
        <UserAvatar user={user} />
        <span className="user-meta">
          <span className="user-name">{user.display_name || user.username}</span>
          <span className="user-dept">{user.department || "—"}</span>
        </span>
        <ChevronDown size={15} className="muted" />
      </button>
      {open ? (
        <div className="user-dropdown">
          <div className="user-dropdown-head">
            <div className="font-semibold">{user.display_name || user.username}</div>
            <div className="muted text-xs">{user.email || user.username}{user.department ? ` · ${user.department}` : ""}</div>
            {user.is_super_admin ? <span className="tag mt-2">超级管理员</span> : null}
          </div>
          <div className="user-dropdown-list">
            <Link className="user-dropdown-item" href="/me/recent" onClick={() => setOpen(false)}><Clock3 size={16} />最近访问</Link>
            <Link className="user-dropdown-item" href="/me/favorites" onClick={() => setOpen(false)}><Star size={16} />我的关注</Link>
            <Link className="user-dropdown-item" href="/me/mcp" onClick={() => setOpen(false)}><Terminal size={16} />MCP 使用</Link>
            {isAdmin ? <Link className="user-dropdown-item" href="/admin" onClick={() => setOpen(false)}><Shield size={16} />管理控制台</Link> : null}
          </div>
          <div className="user-dropdown-foot">
            <button className="user-dropdown-item" onClick={signOut}><LogOut size={16} />登出</button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
