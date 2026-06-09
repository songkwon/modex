import { AdminShell } from "@/components/admin-shell";
import { getModule } from "@/lib/api";

export default async function AdminModuleDetail({ params }: { params: { moduleKey: string } }) {
  const module = await getModule(params.moduleKey);
  const rows = [
    ["模块 Key", module.module_key],
    ["描述", module.description],
    ["分类", module.category_path],
    ["Owner", module.owner_group],
    ["维护人", module.maintainers.join(", ")],
    ["默认文档版本", module.default_version],
    ["工程版本", module.package_version],
    ["Channel", module.channel],
    ["Edition", module.edition],
    ["源码仓库", module.repo_url],
    ["近 7 天阅读", String(module.reads_7d)],
    ["近 30 天阅读", String(module.reads_30d)]
  ];
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
            {module.keywords.map((tag) => <span className="tag" key={tag}>{tag}</span>)}
          </div>
          <div className="empty-state mt-5">
            <div>
              <div className="font-semibold text-slate-700">版本与权限管理</div>
              <p className="mt-2 text-sm">后续接入数据库后在这里维护默认版本、发布权限和阅读权限。</p>
            </div>
          </div>
        </div>
      </section>
    </AdminShell>
  );
}
