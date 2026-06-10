"use client";

import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { SearchPalette, type PaletteScope } from "@/components/search-palette";
import { DocChatPanel } from "@/components/doc-chat-panel";

export type DocChatScope = { moduleKey: string; moduleName: string } | null;

type SearchContextValue = {
  openSearch: () => void;
  setScope: (scope: PaletteScope) => void;
  clearScope: () => void;
  scope: PaletteScope;
  // Doc-level AI chat (only available inside a document).
  docChat: DocChatScope;
  setDocChat: (d: DocChatScope) => void;
  openChat: () => void;
};

const SearchContext = createContext<SearchContextValue>({
  openSearch: () => {},
  setScope: () => {},
  clearScope: () => {},
  scope: {},
  docChat: null,
  setDocChat: () => {},
  openChat: () => {}
});

export const useSearch = () => useContext(SearchContext);

export function SearchProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const [scope, setScopeState] = useState<PaletteScope>({});
  const [docChat, setDocChatState] = useState<DocChatScope>(null);
  const [chatOpen, setChatOpen] = useState(false);

  const openSearch = useCallback(() => setOpen(true), []);
  const setScope = useCallback((s: PaletteScope) => setScopeState(s), []);
  const clearScope = useCallback(() => setScopeState({}), []);
  const setDocChat = useCallback((d: DocChatScope) => {
    setDocChatState(d);
    if (!d) setChatOpen(false);
  }, []);
  const openChat = useCallback(() => setChatOpen(true), []);

  // Cmd/Ctrl+K (and "/") opens the search palette anywhere in the app.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) {
        e.preventDefault();
        setOpen(true);
        return;
      }
      if (e.key === "/" && !open) {
        const el = e.target as HTMLElement;
        const tag = el?.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA" || el?.isContentEditable) return;
        e.preventDefault();
        setOpen(true);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open]);

  return (
    <SearchContext.Provider value={{ openSearch, setScope, clearScope, scope, docChat, setDocChat, openChat }}>
      {children}
      <SearchPalette open={open} onClose={() => setOpen(false)} scope={scope} />
      {docChat ? (
        <DocChatPanel open={chatOpen} onClose={() => setChatOpen(false)} moduleKey={docChat.moduleKey} moduleName={docChat.moduleName} />
      ) : null}
    </SearchContext.Provider>
  );
}
