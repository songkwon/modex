"use client";

import { Sparkles } from "lucide-react";
import { useSearch } from "@/components/search-provider";

// TopbarChatButton appears next to the search box only when inside a document,
// opening the right-side AI conversation about that document.
export function TopbarChatButton() {
  const { docChat, openChat } = useSearch();
  if (!docChat) return null;
  return (
    <button className="button topbar-chat" onClick={openChat} aria-label="与 AI 对话" title="针对本文档询问 AI">
      <Sparkles size={16} />
      <span className="topbar-chat-label">询问 AI</span>
    </button>
  );
}
