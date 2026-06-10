"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { getModule, updateModule } from "@/lib/api";
import type { ModuleInfo } from "@/types/modex";

export default function AdminModuleDetail({ params }: { params: { moduleKey: string } }) {
  const [module, setModule] = useState<ModuleInfo | null>(null);
  const [shownToken, setShownToken] = useState<string>("");
  const [loading, setLoading] = useState(true);

  async function load() {
    const m = await getModule(params.moduleKey);
    setModule(m);
    setLoading(false);
  }

  useEffect(() => {
    load();
  }, [params.moduleKey]);

  if (loading || !module) {
    return <AdminShell title="加载中..." kicker="Module Detail"><div className="panel">加载模块信息...</div></AdminShell>;
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
    ["源码仓库", module.repo_url],
    ["近 7 天阅读", String(module.reads_7d)],
    ["近 30 天阅读", String(module.reads_30d)]
  ];

  const gitlabRows = [
    ["Source Type", module.source_type || "manual"],
    ["GitLab Branch", module.gitlab_branch || "-"],
    ["GitLab Docs Path", module.gitlab_path || "docs (root)"],
    ["Last Synced Commit", module.last_synced_commit || "-"],
    ["Last Synced", module.last_synced_at || "-"],
  ];

  async function generateDeployToken() {
    if (!module) return;
    const newToken = Array.from(crypto.getRandomValues(new Uint8Array(16)))
      .map(b => b.toString(16).padStart(2, "0")).join("");
    try {
      await updateModule(module.module_key, { deploy_token: newToken } as any);
      setShownToken(newToken);
      alert("新 Deploy Token 已生成并保存！请立即复制到 GitLab CI Secret (DOCS_DEPLOY_TOKEN)，此页面不会永久显示完整 token。");
      await load();
    } catch (e) {
      alert("生成失败: " + e);
    }
  }

  async function rotateDeployToken() {
    if (!confirm("确定要轮换 Deploy Token 吗？旧 token 将失效。")) return;
    await generateDeployToken();
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
              <p className="mt-2 text-sm">后续接入数据库后在这里维护默认版本、发布权限和阅读权限。</p>
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
            <button className="button button-primary" onClick={generateDeployToken}>
              生成新 Token
            </button>
            <button className="button" onClick={rotateDeployToken}>
              轮换 Token
            </button>
          </div>
          {shownToken && (
            <div className="mt-2 p-3 bg-yellow-50 border border-yellow-200 rounded text-sm">
              <strong>新 Token（仅此显示一次，请立即复制到 GitLab CI 变量 DOCS_DEPLOY_TOKEN）:</strong>
              <pre className="mt-1 p-2 bg-panel font-mono text-xs break-all">{shownToken}</pre>
            </div>
          )}
          <p className="text-xs muted mt-1">Token 已设置在模块配置中。CI 中使用 X-Modex-Deploy-Token 头或 Authorization Bearer 发送。</p>
        </div>

        <div className="mt-6">
          <h3 className="font-medium mb-2">推荐对接方式（与 Mintlify 类似）</h3>
          <div className="prose prose-sm max-w-none text-sm">
            <p>推荐在<strong>文档仓库的 GitLab CI</strong> 中编译构建，然后通过 <code>docsctl</code> 打包 artifact 并部署到 modex。</p>
            <ul>
              <li>编译在源仓库环境完成（正确 Node 版本、依赖、VitePress/VuePress 插件等）。</li>
              <li>modex 只接收预构建的静态站点 + 结构化元数据（无需在 modex 里实现各种渲染器）。</li>
              <li>一个仓库可以部署多个 Module（不同子目录 → 不同 module_key），分别归属不同“领域”（Category）。</li>
            </ul>
          </div>

          <div className="mt-4 p-4 bg-slate-50 rounded text-xs font-mono overflow-auto">
            <div className="mb-2 text-slate-500"># .gitlab-ci.yml 示例 (rd-doc 仓库)</div>
            <pre>{`include:
  - project: 'devops/docs-ci-templates'
    ref: main
    file: '/templates/docs-deploy.yml'

variables:
  DOCS_MODULE: "rd-standards"          # 或 rd-version-control / rd-workflow 等
  DOCS_VERSION: "latest"
  DOCS_BUILDER: "vitepress"            # 或 vuepress / markdown
  DOCS_SOURCE_DIR: "docs/standard"     # rd-doc 的子目录，映射到 modex 指定位置
  DOCS_DEPLOY_URL: "https://modex.example.com/api/deploy"
  # DOCS_DEPLOY_TOKEN: 从上方生成并复制到 GitLab CI Secret

# 或者自定义 job
deploy-to-modex:
  stage: deploy
  script:
    - npm ci && npm run docs:build   # 你的构建命令
    - docsctl package
    - docsctl deploy
  only:
    - main
  when: on_success`}</pre>
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
