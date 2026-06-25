"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { BookOpen, X } from "lucide-react";
import { getOptionalMe } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import type { User } from "@/types/modex";

function storageKey(user: User) {
  return `modex_welcome_guide_seen:${user.id || user.username || user.email || "anonymous"}`;
}

export function WelcomeGuideToast() {
  const { t } = useI18n();
  const [key, setKey] = useState("");
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    let alive = true;
    getOptionalMe()
      .then((user) => {
        if (!alive || !user || typeof window === "undefined") return;
        const nextKey = storageKey(user);
        if (window.localStorage.getItem(nextKey) === "1") return;
        setKey(nextKey);
        setVisible(true);
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, []);

  function dismiss() {
    if (key && typeof window !== "undefined") {
      window.localStorage.setItem(key, "1");
    }
    setVisible(false);
  }

  if (!visible) return null;

  return (
    <div className="welcome-guide-toast" role="dialog" aria-live="polite" aria-label={t("welcomeGuide.title")}>
      <div className="welcome-guide-toast__icon" aria-hidden>
        <BookOpen size={18} />
      </div>
      <div className="welcome-guide-toast__body">
        <div className="welcome-guide-toast__title">{t("welcomeGuide.title")}</div>
        <p>{t("welcomeGuide.subtitle")}</p>
      </div>
      <div className="welcome-guide-toast__actions">
        <Link className="button button-primary" href="/me/guide" onClick={dismiss}>
          {t("welcomeGuide.action")}
        </Link>
        <button className="button" type="button" onClick={dismiss}>
          {t("welcomeGuide.dismiss")}
        </button>
      </div>
      <button className="welcome-guide-toast__close" type="button" aria-label={t("welcomeGuide.dismiss")} onClick={dismiss}>
        <X size={16} />
      </button>
    </div>
  );
}
