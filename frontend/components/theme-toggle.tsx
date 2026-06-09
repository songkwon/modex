"use client";

import { useEffect, useState } from "react";
import { Monitor, Moon, Sun } from "lucide-react";

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

  const label = choice === "system" ? "跟随系统" : choice === "dark" ? "夜间" : "白天";
  const Icon = choice === "system" ? Monitor : choice === "dark" ? Moon : Sun;

  return (
    <button className="button icon-button" onClick={cycle} title={`主题：${label}（点击切换）`} aria-label={`主题：${label}`}>
      <Icon size={16} />
    </button>
  );
}
