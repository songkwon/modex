"use client";

import type { ModuleInfo } from "@/types/modex";

const FAVORITES_KEY = "modex_favorite_modules";
const RECENTS_KEY = "modex_recent_docs";
const MAX_RECENTS = 30;

export type RecentDoc = {
  doc_id: string;
  title: string;
  module_key: string;
  module_name: string;
  docs_version: string;
  entry_key: string;
  href: string;
  viewed_at: string;
};

function readJSON<T>(key: string, fallback: T): T {
  if (typeof window === "undefined") return fallback;
  try {
    const raw = window.localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : fallback;
  } catch {
    return fallback;
  }
}

function writeJSON<T>(key: string, value: T) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(key, JSON.stringify(value));
}

export function favoriteModuleKeys(): string[] {
  return readJSON<string[]>(FAVORITES_KEY, []);
}

export function isFavoriteModule(moduleKey: string): boolean {
  return favoriteModuleKeys().includes(moduleKey);
}

export function setFavoriteModule(moduleKey: string, favorite: boolean): string[] {
  const next = new Set(favoriteModuleKeys());
  if (favorite) {
    next.add(moduleKey);
  } else {
    next.delete(moduleKey);
  }
  const keys = Array.from(next);
  writeJSON(FAVORITES_KEY, keys);
  window.dispatchEvent(new CustomEvent("modex:favorites-changed", { detail: keys }));
  return keys;
}

export function favoriteModulesFrom(allModules: ModuleInfo[]): ModuleInfo[] {
  const order = favoriteModuleKeys();
  const byKey = new Map(allModules.map((module) => [module.module_key, module]));
  return order.map((key) => byKey.get(key)).filter(Boolean) as ModuleInfo[];
}

export function recentDocs(): RecentDoc[] {
  return readJSON<RecentDoc[]>(RECENTS_KEY, []);
}

export function recordRecentDoc(doc: Omit<RecentDoc, "viewed_at">) {
  const next: RecentDoc = { ...doc, viewed_at: new Date().toISOString() };
  const deduped = recentDocs().filter((item) => item.doc_id !== doc.doc_id);
  writeJSON(RECENTS_KEY, [next, ...deduped].slice(0, MAX_RECENTS));
}
