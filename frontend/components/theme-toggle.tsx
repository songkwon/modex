"use client";

import { useEffect, useState } from "react";
import { Monitor, Moon, Sun } from "lucide-react";
import { useI18n } from "@/lib/i18n";

type ThemeChoice = "light" | "dark" | "system";
const KEY = "modex_theme";
const order: ThemeChoice[] = ["system", "light", "dark"];

function systemDark() {
  return typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function apply(choice: ThemeChoice) {
  const dark = choice === "dark" || (choice === "system" && systemDark());
  document.documentElement.dataset.theme = dark ? "dark" : "light";
}

export function ThemeToggle() {
  const { t } = useI18n();
  const [choice, setChoice] = useState<ThemeChoice>("system");

  useEffect(() => {
    const stored = (localStorage.getItem(KEY) as ThemeChoice) || "system";
    setChoice(stored);
    apply(stored);
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      if ((localStorage.getItem(KEY) as ThemeChoice) === "system" || !localStorage.getItem(KEY)) apply("system");
    };
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);

  function cycle() {
    const next = order[(order.indexOf(choice) + 1) % order.length];
    setChoice(next);
    localStorage.setItem(KEY, next);
    apply(next);
  }

  const label = choice === "system" ? t("legacy.217cfe7db1e3") : choice === "dark" ? t("legacy.df2cedace72a") : t("legacy.6fa7e15a01ab");
  const Icon = choice === "system" ? Monitor : choice === "dark" ? Moon : Sun;

  return (
    <button className="button icon-button" onClick={cycle} title={t("legacy.fc175993cd37", { value1: label })} aria-label={t("legacy.fc83a6ca6db4", { value1: label })}>
      <Icon size={18} />
    </button>
  );
}
