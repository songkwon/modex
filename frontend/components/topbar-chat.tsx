"use client";

import { Sparkles } from "lucide-react";
import { useSearch } from "@/components/search-provider";
import { useI18n } from "@/lib/i18n";

// TopbarChatButton appears next to the search box only when inside a document,
// opening the right-side AI conversation about that document.
export function TopbarChatButton() {
  const { t } = useI18n();
  const { docChat, openChat } = useSearch();
  if (!docChat) return null;
  return (
    <button className="button topbar-chat" onClick={openChat} aria-label={t("nav.askAI")} title={t("nav.askAITitle")}>
      <Sparkles size={16} />
      <span className="topbar-chat-label">{t("nav.askAI")}</span>
    </button>
  );
}
