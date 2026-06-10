"use client";

import { useEffect } from "react";
import { useSearch } from "@/components/search-provider";

// DocScope registers the current document with the global search/chat chrome:
// the ⌘K palette searches only this module, and the topbar shows the AI chat
// button. It renders nothing and cleans up on unmount.
export function DocScope({ moduleKey, moduleName }: { moduleKey: string; moduleName: string }) {
  const { setScope, clearScope, setDocChat } = useSearch();
  useEffect(() => {
    setScope({ moduleKey, label: moduleName });
    setDocChat({ moduleKey, moduleName });
    return () => {
      clearScope();
      setDocChat(null);
    };
  }, [moduleKey, moduleName, setScope, clearScope, setDocChat]);
  return null;
}
