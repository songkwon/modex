"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";
import zhCN from "@/messages/zh-CN.json";
import enUS from "@/messages/en-US.json";

export const locales = ["zh-CN", "en-US"] as const;
export type Locale = (typeof locales)[number];
type Messages = Record<string, string>;

const messages: Record<Locale, Messages> = {
  "zh-CN": zhCN,
  "en-US": enUS
};

const localeLabels: Record<Locale, string> = {
  "zh-CN": "中文",
  "en-US": "English"
};

type I18nValue = {
  locale: Locale;
  locales: readonly Locale[];
  localeLabel: (locale: Locale) => string;
  setLocale: (locale: Locale) => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
};

const I18nContext = createContext<I18nValue | null>(null);

function normalizeLocale(value: string | null | undefined): Locale {
  if (!value) return "zh-CN";
  const exact = locales.find((locale) => locale === value);
  if (exact) return exact;
  const lower = value.toLowerCase();
  if (lower.startsWith("en")) return "en-US";
  return "zh-CN";
}

function interpolate(template: string, vars?: Record<string, string | number>) {
  if (!vars) return template;
  return template.replace(/\{\{(\w+)\}\}/g, (_, key: string) => String(vars[key] ?? ""));
}

const legacyKeys = Object.keys(zhCN).filter((key) => key.startsWith("legacy."));
const legacySourceToKey = new Map(legacyKeys.map((key) => [zhCN[key as keyof typeof zhCN], key]));
const originalText = new WeakMap<Text, string>();
const originalAttributes = new WeakMap<Element, Map<string, string>>();
const translatedAttributes = ["aria-label", "placeholder", "title"];

// Resolve a rendered literal back to its legacy.* key (direct, or via a
// {{valueN}} pattern). Returns null for anything the legacy catalog does not
// own -- in particular text rendered through t(), which React controls and the
// observer must never touch.
function legacyMatch(value: string): { key: string; captures: string[] } | null {
  const direct = legacySourceToKey.get(value);
  if (direct) return { key: direct, captures: [] };
  for (const [source, key] of legacySourceToKey) {
    if (!source.includes("{{value")) continue;
    const parts = source.split(/\{\{value\d+\}\}/g);
    const pattern = new RegExp(`^${parts.map(escapeRegExp).join("(.+?)")}$`);
    const match = value.match(pattern);
    if (match) return { key, captures: match.slice(1) };
  }
  return null;
}

function isLegacySource(value: string) {
  return legacyMatch(value) !== null;
}

function translateLegacyLiteral(value: string, locale: Locale) {
  if (locale === "zh-CN") return value;
  const matched = legacyMatch(value);
  if (!matched) return value;
  let translated = messages[locale][matched.key] || value;
  matched.captures.forEach((capture, index) => {
    translated = translated.replaceAll(`{{value${index + 1}}}`, capture);
  });
  return translated;
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function applyLegacyTranslations(root: ParentNode, locale: Locale) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  let node = walker.nextNode() as Text | null;
  while (node) {
    const parent = node.parentElement;
    if (parent && !parent.closest("pre, code, script, style, textarea, [data-i18n-ignore]")) {
      const source = originalText.get(node) ?? node.data;
      const trimmed = source.trim();
      // Only manage nodes whose original text is a legacy literal. Text rendered
      // by t() is React-owned; rewriting it here would clobber live translations
      // (the bug this guard fixes). Once a node is adopted we keep managing it so
      // switching back to zh-CN restores the original.
      if (originalText.has(node) || isLegacySource(trimmed)) {
        originalText.set(node, source);
        const translated = translateLegacyLiteral(trimmed, locale);
        const next = source.replace(trimmed, translated);
        if (node.data !== next) node.data = next;
      }
    }
    node = walker.nextNode() as Text | null;
  }
  const elements = root instanceof Element ? [root, ...root.querySelectorAll("*")] : [...root.querySelectorAll("*")];
  for (const element of elements) {
    let saved = originalAttributes.get(element);
    if (!saved) {
      saved = new Map();
      originalAttributes.set(element, saved);
    }
    for (const attribute of translatedAttributes) {
      const current = element.getAttribute(attribute);
      if (current === null) continue;
      if (!saved.has(attribute)) {
        // Same rule as text nodes: leave t()-provided attribute values alone.
        if (!isLegacySource(current.trim())) continue;
        saved.set(attribute, current);
      }
      const next = translateLegacyLiteral(saved.get(attribute) || current, locale);
      if (current !== next) element.setAttribute(attribute, next);
    }
  }
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("zh-CN");

  useEffect(() => {
    const stored = window.localStorage.getItem("modex_locale");
    setLocaleState(normalizeLocale(stored || window.navigator.language));
  }, []);

  const value = useMemo<I18nValue>(() => {
    const setLocale = (next: Locale) => {
      setLocaleState(next);
      window.localStorage.setItem("modex_locale", next);
      document.documentElement.lang = next;
    };
    const t = (key: string, vars?: Record<string, string | number>) => {
      const dict = messages[locale] || messages["zh-CN"];
      return interpolate(dict[key] || messages["zh-CN"][key] || key, vars);
    };
    return {
      locale,
      locales,
      localeLabel: (l) => localeLabels[l],
      setLocale,
      t
    };
  }, [locale]);

  useEffect(() => {
    document.documentElement.lang = locale;
    const observeOptions: MutationObserverInit = { childList: true, characterData: true, subtree: true };
    let observer: MutationObserver;
    // Disconnect while we mutate the DOM so the observer never reacts to its
    // own writes. A boolean guard does not work here because MutationObserver
    // callbacks are delivered asynchronously (after the guard is reset), which
    // previously caused an infinite re-translation loop and pinned the CPU.
    const apply = (root: ParentNode) => {
      observer.disconnect();
      applyLegacyTranslations(root, locale);
      observer.takeRecords();
      observer.observe(document.body, observeOptions);
    };
    observer = new MutationObserver((records) => {
      const roots = new Set<ParentNode>();
      for (const record of records) {
        if (record.type === "characterData" && record.target.parentNode) roots.add(record.target.parentNode);
        for (const added of record.addedNodes) {
          if (added instanceof Element) roots.add(added);
          else if (added.parentNode) roots.add(added.parentNode);
        }
      }
      if (roots.size === 0) return;
      observer.disconnect();
      for (const root of roots) applyLegacyTranslations(root, locale);
      observer.takeRecords();
      observer.observe(document.body, observeOptions);
    });
    apply(document.body);
    return () => observer.disconnect();
  }, [locale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used inside I18nProvider");
  }
  return ctx;
}
