"use client";

import { useState } from "react";
import { RefreshCw } from "lucide-react";
import { reindexEmbeddings, reindexSearch } from "@/lib/api";

export function ReindexControls() {
  const [busy, setBusy] = useState<string>("");
  const [result, setResult] = useState<string>("");

  async function run(kind: "search" | "embeddings") {
    setBusy(kind);
    setResult("");
    try {
      const res = kind === "search" ? await reindexSearch() : await reindexEmbeddings();
      setResult(`${kind}: ${JSON.stringify(res)}`);
    } catch (e) {
      setResult(String(e));
    } finally {
      setBusy("");
    }
  }

  return (
    <section className="panel">
      <h2 className="font-semibold">索引维护</h2>
      <p className="muted mt-1 text-sm">重建关键词索引与语义向量缓存（发布新文档后可手动触发）。</p>
      <div className="mt-3 flex flex-wrap gap-2">
        <button className="button" disabled={busy !== ""} onClick={() => run("search")}>
          <RefreshCw size={16} />{busy === "search" ? "重建中…" : "重建搜索索引"}
        </button>
        <button className="button" disabled={busy !== ""} onClick={() => run("embeddings")}>
          <RefreshCw size={16} />{busy === "embeddings" ? "重建中…" : "重建向量索引"}
        </button>
      </div>
      {result ? <pre className="muted mt-3 overflow-auto text-xs">{result}</pre> : null}
    </section>
  );
}
