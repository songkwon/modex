"use client";

import { useEffect, useState } from "react";
import { RefreshCw } from "lucide-react";
import { getMe, reindexEmbeddings, reindexSearch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

export function ReindexControls() {
  const { t } = useI18n();
  const [busy, setBusy] = useState<string>("");
  const [result, setResult] = useState<string>("");
  const [isSuper, setIsSuper] = useState<boolean | null>(null);

  useEffect(() => {
    getMe().then((me) => setIsSuper(!!me.is_super_admin)).catch(() => setIsSuper(false));
  }, []);

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

  // Index maintenance is a platform-wide operation; hide it from team admins.
  if (!isSuper) return null;

  return (
    <section className="panel">
      <h2 className="font-semibold">{t("legacy.676706b517e5")}</h2>
      <p className="muted mt-1 text-sm">{t("legacy.b5ef8194c550")}</p>
      <div className="mt-3 flex flex-wrap gap-2">
        <button className="button" disabled={busy !== ""} onClick={() => run("search")}>
          <RefreshCw size={16} />{busy === "search" ? t("legacy.977350126f6b") : t("legacy.a71015b71f25")}
        </button>
        <button className="button" disabled={busy !== ""} onClick={() => run("embeddings")}>
          <RefreshCw size={16} />{busy === "embeddings" ? t("legacy.977350126f6b") : t("legacy.6abfc57f651e")}
        </button>
      </div>
      {result ? <pre className="muted mt-3 overflow-auto text-xs">{result}</pre> : null}
    </section>
  );
}
