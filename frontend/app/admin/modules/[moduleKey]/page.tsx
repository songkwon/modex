"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { getModule, rotateDeployToken } from "@/lib/api";
import type { ModuleInfo } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

export default function AdminModuleDetail({ params }: { params: Promise<{ moduleKey: string }> }) {
  const { t } = useI18n();
  const [module, setModule] = useState<ModuleInfo | null>(null);
  const [shownToken, setShownToken] = useState<string>("");
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load(key: string) {
    setError(null);
    try {
      const m = await getModule(key);
      setModule(m);
    } catch (e) {
      setError(t("legacy.d750df5c5773") + (e instanceof Error ? e.message : String(e)));
      setModule(null);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let cancelled = false;
    params.then(({ moduleKey: key }) => {
      if (cancelled) return;
      load(key);
    });
    return () => {
      cancelled = true;
    };
  }, [params]);

  if (loading) {
    return <AdminShell title={t("legacy.9dc0825fba54")} kicker="Module Detail"><div className="panel">{t("legacy.4aec040a6fa7")}</div></AdminShell>;
  }

  if (error || !module) {
    return (
      <AdminShell title={t("legacy.3f6c407778bd")} kicker="Module Detail">
        <div className="panel text-red-600">{error || t("legacy.9d1e16203035")}</div>
      </AdminShell>
    );
  }

  const rows = [
    [t("legacy.5f8716f9d7f6"), module.module_key],
    [t("legacy.dc2ba467fc7a"), module.description],
    [t("legacy.515559957fd3"), module.category_path],
    ["Owner", module.owner_group],
    [t("legacy.372558ec3231"), module.maintainers?.join(", ") || "-"],
    [t("legacy.c68acbc684f6"), module.default_version],
    [t("legacy.ff9af643cee0"), module.package_version],
    ["Channel", module.channel],
    ["Edition", module.edition],
    [t("legacy.908a5a92dc38"), module.repo_url]
  ];

  const gitlabRows = [
    ["Source Type", module.source_type || "manual"],
    ["GitLab Branch", module.gitlab_branch || "-"],
    ["GitLab Docs Path", module.gitlab_path || "docs (root)"],
    ["Last Synced Commit", module.last_synced_commit || "-"],
    ["Last Synced", module.last_synced_at || "-"],
  ];

  async function handleRotateDeployToken() {
    if (!module) return;
    if (!confirm(t("legacy.24f2a74a81e6"))) return;
    try {
      const res = await rotateDeployToken(module.module_key);
      setShownToken(res.deploy_token);
      setCopied(false);
      alert(t("legacy.068c38574ee2"));
      await load(module.module_key);
    } catch (e) {
      alert(t("legacy.4424caf30887") + (e instanceof Error ? e.message : String(e)));
    }
  }

  async function copyToken() {
    if (!shownToken) return;
    try {
      await navigator.clipboard.writeText(shownToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      alert(t("legacy.69ba230370ea"));
    }
  }

  return (
    <AdminShell title={module.name} kicker="Module Detail" description={t("legacy.b0b986d23c77")}>
      <section className="split-panel">
        <div className="panel">
          <div className="section-heading">
            <h2>{t("legacy.3d74fbdde081")}</h2>
            <span className="tag">{module.status}</span>
          </div>
          <dl className="detail-list">
            {rows.map(([label, value]) => (
              <div key={label}>
                <dt>{label}</dt>
                <dd className={value === module.repo_url ? "break-all" : ""}>{value}</dd>
              </div>
            ))}
          </dl>
        </div>
        <div className="panel">
          <div className="section-heading">
            <h2>{t("legacy.1d0fd5f9336d")}</h2>
          </div>
          <div className="flex flex-wrap gap-2">
            {module.keywords?.map((tag) => <span className="tag" key={tag}>{tag}</span>)}
          </div>
          <div className="empty-state mt-5">
            <div>
              <div className="font-semibold text-foreground">{t("legacy.31fad99fe77f")}</div>
              <p className="mt-2 text-sm">{t("legacy.400e061fb072")}</p>
            </div>
          </div>
        </div>
      </section>

      {/* GitLab 集成 & 部署 */}
      <section className="panel">
        <div className="section-heading">
          <h2>{t("legacy.270d01614c5e")}</h2>
          <span className="tag">{t("legacy.0c554a3f5c2c")}</span>
        </div>

        <div className="mt-4">
          <h3 className="font-medium mb-2">{t("legacy.9c794f997d82")}</h3>
          <dl className="detail-list text-sm">
            {gitlabRows.map(([label, value]) => (
              <div key={label}>
                <dt>{label}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
        </div>

        {/* Deploy Token Management - direct in UI */}
        <div className="mt-6">
          <h3 className="font-medium mb-2">{t("legacy.3c60fc5e0ca8")}</h3>
          <div className="flex items-center gap-2">
            <button className="button button-primary" onClick={handleRotateDeployToken}>
              {t("legacy.127dffaa3a33")}
            </button>
          </div>
          {shownToken && (
            <div className="mt-3 p-3 bg-yellow-50 border border-yellow-200 rounded text-sm">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <strong>{t("legacy.1826efd01b78")}</strong>
                </div>
                <button
                  className="shrink-0 px-2 py-1 text-xs rounded border border-yellow-300 bg-white hover:bg-yellow-100 transition"
                  onClick={copyToken}
                >
                  {copied ? t("legacy.8f6f8d979c98") : t("legacy.63d90d977348")}
                </button>
              </div>
              <pre className="mt-2 p-2 bg-panel font-mono text-xs break-all select-all">{shownToken}</pre>
            </div>
          )}
          <p className="text-xs muted mt-2">
            {t("legacy.25d24ce366f4")}
            {module?.deploy_token_set
              ? t("legacy.b3816850e089")
              : t("legacy.de3484362442")}
          </p>
        </div>

        <div className="mt-6">
          <h3 className="font-medium mb-2">{t("legacy.438d7c5580dc")}</h3>
          <div className="prose prose-sm max-w-none text-sm">
            <p>{t("legacy.828e9f69c5a0")}<strong>{t("legacy.e00ceaf58a80")}</strong> {t("legacy.0f38667b5608")} <code>docsctl deploy</code>{t("legacy.0612fb08e519")}</p>
            <ul>
              <li>{t("legacy.ea50aa2b959c")}</li>
              <li>{t("legacy.212e850d43e8")}</li>
              <li>{t("legacy.d0709bf1ecda")}</li>
            </ul>
          </div>

          <div className="mt-4 p-4 bg-slate-50 rounded text-xs font-mono overflow-auto">
            <div className="mb-2 text-slate-500"># .gitlab-ci.yml 示例 (rd-doc 仓库)</div>
            <pre>{`include:
  - remote: "https://raw.githubusercontent.com/songkwon/modex/main/deploy/ci/modex-docs.gitlab-ci.yml"

variables:
  MODEX_MODULE_KEY: "rd-standards"     # 必须等于 modex 后台文档源 key
  DOCS_VERSION: "latest"
  DOCS_BUILDER: "vitepress"            # vitepress/vuepress/fumadocs/docusaurus/mkdocs/honkit/gitbook/markdown/static
  DOCS_SOURCE_DIR: "docs/standard"     # rd-doc 的子目录，映射到 modex 指定位置
  DOCS_BUILD: "npm ci && npm run docs:build"
  DOCS_OUTPUT: "docs/.vitepress/dist"
  MODEX_DEPLOY_URL: "https://modex.example.com/api/deploy"

# DOCS_DEPLOY_TOKEN: 从上方生成并复制到 GitLab CI Variables（Masked + Protected）`}</pre>
          </div>

          <p className="mt-3 text-sm text-slate-600">
            {t("legacy.4d21db278314")}
          </p>

          <div className="mt-4 text-xs">
            <strong>{t("legacy.f538712c0912")}</strong>{t("legacy.b2bdfb3ec9c8")}<br/>
            {t("legacy.f78195bf4658")} <code>rd-standard</code>、<code>rd-version-control</code> {t("legacy.c2df94b84b3c")}
          </div>
        </div>
      </section>
    </AdminShell>
  );
}
