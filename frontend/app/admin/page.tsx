import Link from "next/link";
import { BarChart3, Boxes, FolderTree, History, MessageSquareText, Search, Users } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { ReindexControls } from "@/components/reindex-controls";

const links = [
  ["/admin/categories", "分类管理", "维护层级分类、排序和状态", FolderTree],
  ["/admin/modules", "模块管理", "查看模块 Owner、版本、仓库来源", Boxes],
  ["/admin/users", "用户管理", "管理用户、用户组、角色和状态", Users],
  ["/admin/releases", "发布记录", "追踪 Pipeline、构建与发布结果", History],
  ["/admin/analytics", "阅读统计", "查看 PV、UV 和热门文档", BarChart3],
  ["/admin/search-logs", "搜索日志", "分析查询词、命中数和点击", Search],
  ["/admin/mcp-logs", "MCP 日志", "查看 AI 工具读取文档记录", MessageSquareText]
] as const;

export default function AdminPage() {
  return (
    <AdminShell title="治理控制台" kicker="Modex Admin" description="管理分类、模块、发布记录、搜索与 MCP 调用记录。页面结构按后续数据库接入预留。">
      <section className="card-grid">
        {links.map(([href, label, desc, Icon]) => (
          <Link className="card module-card" href={href} key={href}>
            <div className="flex items-center justify-between">
              <Icon size={20} className="text-blue-600" />
              <span className="tag">Admin</span>
            </div>
            <h2 className="mt-4 text-lg font-semibold">{label}</h2>
            <p className="muted mt-2 text-sm leading-6">{desc}</p>
          </Link>
        ))}
      </section>
      <ReindexControls />
    </AdminShell>
  );
}
