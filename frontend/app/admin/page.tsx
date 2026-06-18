import Link from "next/link";
import { ArrowRight, Boxes, FolderTree, History, Link2, MessageSquareText, Search, Settings, Users, UsersRound } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { ReindexControls } from "@/components/reindex-controls";

const links = [
  ["/admin/categories", "分类管理", "维护层级分类、排序、状态与负责团队", FolderTree],
  ["/admin/teams", "团队管理", "维护文档团队的负责人与成员，并为分类指定负责方", UsersRound],
  ["/admin/modules", "文档源", "接入文档仓库、绑定分类、生成 Deploy Token", Boxes],
  ["/admin/users", "用户管理", "管理用户资料、状态与权限", Users],
  ["/admin/settings", "模型设置", "对接对话 / 向量 / 重排序模型，启用 AI 问答", Settings],
  ["/admin/connected-apps", "应用链接", "注册外部应用，通过 OAuth 授权访问 Modex API 与 MCP", Link2],
  ["/admin/releases", "发布记录", "追踪每次文档发布的来源、构建与状态", History],
  ["/admin/search-logs", "搜索日志", "分析查询词、命中数与点击", Search],
  ["/admin/mcp-logs", "MCP 日志", "查看 AI 工具读取文档的记录", MessageSquareText],
] as const;

export default function AdminPage() {
  return (
    <AdminShell title="管理后台" kicker="Admin" description="管理分类、团队、用户、文档源、模型与发布。">
      <section className="card-grid">
        {links.map(([href, label, desc, Icon]) => (
          <Link className="admin-entry" href={href} key={href}>
            <span className="admin-entry-icon">
              <Icon size={20} />
            </span>
            <div className="admin-entry-body">
              <h2>{label}</h2>
              <p>{desc}</p>
            </div>
            <ArrowRight size={16} className="admin-entry-arrow" />
          </Link>
        ))}
      </section>
      <ReindexControls />
    </AdminShell>
  );
}
