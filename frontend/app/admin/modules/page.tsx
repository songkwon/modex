import Link from "next/link";
import { AdminShell } from "@/components/admin-shell";
import { getModules } from "@/lib/api";

export default async function AdminModulesPage() {
  const modules = await getModules();
  return (
    <AdminShell title="模块管理" kicker="Docs Module" description="模块展示信息、Owner、默认版本、权限和发布治理字段由 Registry 管理。">
      <section className="panel">
        <table className="data-table">
          <thead>
            <tr>
              <th>模块</th>
              <th>分类</th>
              <th>Owner</th>
              <th>默认版本</th>
              <th>工程版本</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            {modules.map((m) => (
              <tr key={m.module_key}>
                <td>
                  <Link className="font-semibold text-blue-600" href={`/admin/modules/${m.module_key}`}>{m.name}</Link>
                  <div className="muted text-xs">{m.module_key}</div>
                </td>
                <td>{m.category_path}</td>
                <td>{m.owner_group}</td>
                <td><span className="tag">{m.default_version}</span></td>
                <td>{m.package_version}</td>
                <td><span className="status-dot mr-2" />{m.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </AdminShell>
  );
}
