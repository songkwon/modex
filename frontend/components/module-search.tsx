"use client";

import { useState } from "react";
import { Search, X } from "lucide-react";
import { DocSearch } from "@/components/doc-search";

// ModuleSearch shows a search icon inside a module's docs; expanding it reveals
// a search scoped to the current module, plus the same Ask-AI conversation.
export function ModuleSearch({ moduleKey, moduleName }: { moduleKey: string; moduleName: string }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="module-search">
      <button className="button" onClick={() => setOpen((v) => !v)} title={`在「${moduleName}」内搜索`}>
        {open ? <X size={16} /> : <Search size={16} />}本模块搜索
      </button>
      {open ? (
        <div className="module-search-pop">
          <DocSearch moduleKey={moduleKey} placeholder={`在「${moduleName}」内搜索，或向 AI 提问…`} />
        </div>
      ) : null}
    </div>
  );
}
