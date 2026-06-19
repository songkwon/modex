"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Terminal, Shield, LogOut, LogIn, ChevronDown, FileText } from "lucide-react";
import { getAuthConfig, getOptionalMe, logout, mockLogin } from "@/lib/api";
import { identify } from "@/lib/analytics";
import { useI18n } from "@/lib/i18n";
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
  const { locale, locales, localeLabel, setLocale, t } = useI18n();
  const [user, setUser] = useState<User | null>(null);
  const [cfg, setCfg] = useState<AuthConfig | null>(null);
  const [open, setOpen] = useState(false);
  const [checkedMe, setCheckedMe] = useState(false);
  const autoLoginStarted = useRef(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    getOptionalMe()
      .then((u) => {
        if (!u) {
          setUser(null);
          return;
        }
        setUser(u);
        identify({
          id: u.id || u.username || u.email,
          displayName: u.display_name,
          email: u.email,
          department: u.department,
          groups: u.groups,
        });
      })
      .catch(() => setUser(null))
      .finally(() => setCheckedMe(true));
    getAuthConfig().then(setCfg).catch(() => {});
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  useEffect(() => {
    if (!checkedMe || user || !cfg?.auto_login || autoLoginStarted.current) return;
    if (typeof window !== "undefined" && window.sessionStorage.getItem("modex_manual_logout") === "1") return;
    autoLoginStarted.current = true;
    if (cfg.auth_mode === "oidc") {
      if (cfg.login_url) window.location.href = cfg.login_url;
      return;
    }
    mockLogin()
      .then((res) => setUser(res.user))
      .catch(() => {});
  }, [cfg, checkedMe, user]);

  async function login() {
    if (typeof window !== "undefined") window.sessionStorage.removeItem("modex_manual_logout");
    if (cfg?.auth_mode === "oidc") {
      if (cfg.login_url) window.location.href = cfg.login_url;
      else alert(t("user.oidcNotConfigured"));
      return;
    }
    const res = await mockLogin();
    setUser(res.user);
  }

  async function signOut() {
    await logout();
    if (typeof window !== "undefined") window.sessionStorage.setItem("modex_manual_logout", "1");
    setUser(null);
    setOpen(false);
  }

  if (!user) {
    return (
      <button className="button button-primary" onClick={login}>
        <LogIn size={16} />{t("user.login")}
      </button>
    );
  }

  const isAdmin = user.is_super_admin || user.is_team_admin || (user.roles || []).includes("admin");

  return (
    <div className="user-menu" ref={ref}>
      <button className="user-menu-trigger" onClick={() => setOpen((v) => !v)} aria-haspopup="menu" aria-expanded={open}>
        <UserAvatar user={user} />
        <span className="user-meta">
          <span className="user-name">{user.display_name || user.username}</span>
          <span className="user-dept">{user.department || t("common.emptyDash")}</span>
        </span>
        <ChevronDown size={15} className="muted" />
      </button>
      {open ? (
        <div className="user-dropdown">
          <div className="user-dropdown-head">
            <div className="font-semibold">{user.display_name || user.username}</div>
            <div className="muted text-xs">{user.email || user.username}{user.department ? ` · ${user.department}` : ""}</div>
            {user.is_super_admin ? <span className="tag mt-2">{t("user.superAdmin")}</span> : null}
            <div className="user-locale-row" aria-label={t("nav.language")}>
              <span>{t("nav.language")}</span>
              <span className="user-locale-options">
                {locales.map((item) => (
                  <button
                    key={item}
                    className={item === locale ? "active" : ""}
                    onClick={() => setLocale(item)}
                    type="button"
                  >
                    {localeLabel(item)}
                  </button>
                ))}
              </span>
            </div>
          </div>
          <div className="user-dropdown-list">
            <Link className="user-dropdown-item" href="/me/guide" onClick={() => setOpen(false)}><FileText size={16} />{t("user.projectGuide")}</Link>
            <Link className="user-dropdown-item" href="/me/mcp" onClick={() => setOpen(false)}><Terminal size={16} />{t("user.mcp")}</Link>
            {isAdmin ? <Link className="user-dropdown-item" href="/admin" onClick={() => setOpen(false)}><Shield size={16} />{t("user.admin")}</Link> : null}
          </div>
          <div className="user-dropdown-foot">
            <button className="user-dropdown-item" onClick={signOut}><LogOut size={16} />{t("user.logout")}</button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
