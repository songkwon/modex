"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { getModule, rotateDeployToken } from "@/lib/api";
import type { ModuleInfo } from "@/types/modex";

export default function AdminModuleDetail({ params }: { params: Promise<{ moduleKey: string }> }) {
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
      setError("加载模块信息失败: " + (e instanceof Error ? e.message : String(e)));
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
    return <AdminShell title="加载中..." kicker="Module Detail"><div className="panel">加载模块信息...</div></AdminShell>;
  }

  if (error || !module) {
    return (
      <AdminShell title="出错了" kicker="Module Detail">
        <div className="panel text-red-600">{error || "未找到模块"}</div>
      </AdminShell>
    );
  }

  const rows = [
    ["模块 Key", module.module_key],
    ["描述", module.description],
    ["分类", module.category_path],
    ["Owner", module.owner_group],
    ["维护人", module.maintainers?.join(", ") || "-"],
    ["默认文档版本", module.default_version],
    ["工程版本", module.package_version],
    ["Channel", module.channel],
    ["Edition", module.edition],
    ["源码仓库", module.repo_url]
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
    if (!confirm("确定要生成新的 Deploy Token 吗？旧 token 将立即失效。")) return;
    try {
      const res = await rotateDeployToken(module.module_key);
      setShownToken(res.deploy_token);
      setCopied(false);
      alert("新 Deploy Token 已生成并保存！请立即复制到 GitLab CI Secret (DOCS_DEPLOY_TOKEN)，此页面不会永久显示完整 token。");
      await load(module.module_key);
    } catch (e) {
      alert("生成失败: " + (e instanceof Error ? e.message : String(e)));
    }
  }

  async function copyToken() {
    if (!shownToken) return;
    try {
      await navigator.clipboard.writeText(shownToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      alert("复制失败，请手动选中下方 token 复制");
    }
  }

  return (
    <AdminShell title={module.name} kicker="Module Detail" description="模块治理字段、版本入口和后续权限策略都从这里收口。">
      <section className="split-panel">
        <div className="panel">
          <div className="section-heading">
            <h2>治理字段</h2>
            <span className="tag">{module.status}</span>
          </div>
          <dl className="detail-list">
            {rows.map(([label, value]) => (
              <div key={label}>
                <dt>{label}</dt>
                <dd className={label === "源码仓库" ? "break-all" : ""}>{value}</dd>
              </div>
            ))}
          </dl>
        </div>
        <div className="panel">
          <div className="section-heading">
            <h2>标签</h2>
          </div>
          <div className="flex flex-wrap gap-2">
            {module.keywords?.map((tag) => <span className="tag" key={tag}>{tag}</span>)}
          </div>
          <div className="empty-state mt-5">
            <div>
              <div className="font-semibold text-foreground">版本与权限管理</div>
              <p className="mt-2 text-sm">在这里维护默认版本、发布权限和阅读权限。</p>
            </div>
          </div>
        </div>
      </section>

      {/* GitLab 集成 & 部署 */}
      <section className="panel">
        <div className="section-heading">
          <h2>GitLab 集成 & 部署</h2>
          <span className="tag">CI 驱动</span>
        </div>

        <div className="mt-4">
          <h3 className="font-medium mb-2">来源配置</h3>
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
          <h3 className="font-medium mb-2">Deploy Token（用于 GitLab CI 安全部署）</h3>
          <div className="flex items-center gap-2">
            <button className="button button-primary" onClick={handleRotateDeployToken}>
              生成 / 轮换 Token
            </button>
          </div>
          {shownToken && (
            <div className="mt-3 p-3 bg-yellow-50 border border-yellow-200 rounded text-sm">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <strong>新 Token（仅此显示一次，请立即复制到 GitLab CI 变量 DOCS_DEPLOY_TOKEN）:</strong>
                </div>
                <button
                  className="shrink-0 px-2 py-1 text-xs rounded border border-yellow-300 bg-white hover:bg-yellow-100 transition"
                  onClick={copyToken}
                >
                  {copied ? "已复制" : "复制"}
                </button>
              </div>
              <pre className="mt-2 p-2 bg-panel font-mono text-xs break-all select-all">{shownToken}</pre>
            </div>
          )}
          <p className="text-xs muted mt-2">
            CI 中使用 X-Modex-Deploy-Token 头或 Authorization Bearer 发送。
            {module?.deploy_token_set
              ? " 当前已设置 Deploy Token（出于安全不再显示明文）。"
              : " 当前尚未设置 Deploy Token。"}
          </p>
        </div>

        <div className="mt-6">
          <h3 className="font-medium mb-2">推荐对接方式（与 Mintlify 类似）</h3>
          <div className="prose prose-sm max-w-none text-sm">
            <p>推荐在<strong>文档仓库的 GitLab CI</strong> 中执行 <code>docsctl deploy</code>，自动完成校验、构建、打包和上传。</p>
            <ul>
              <li>编译在源仓库环境完成（正确 Node/Python 版本、依赖和文档框架插件）。</li>
              <li>modex 只接收预构建的静态站点 + 结构化元数据（无需在 modex 里实现各种渲染器）。</li>
              <li>一个仓库可以部署多个 Module（不同子目录 → 不同 module_key），分别归属不同“领域”（Category）。</li>
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
            部署后文档会自动归属到该 Module 关联的“领域”（Category 树中的指定位置）。
          </p>

          <div className="mt-4 text-xs">
            <strong>rd-doc 例子</strong>：仓库根有 docs/standard/、docs/tools/version-control/ 等。<br/>
            可以用不同 job（或 matrix）分别部署为 <code>rd-standard</code>、<code>rd-version-control</code> 等 Module，然后在 modex 里把它们分配到对应的领域（标准规范、工具规范...）。
          </div>
        </div>
      </section>
    </AdminShell>
  );
}
