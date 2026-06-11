import Link from "next/link";
import { ArrowRight, BarChart3, Boxes, FolderTree, History, MessageSquareText, Search, Settings, Users, UsersRound } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { ReindexControls } from "@/components/reindex-controls";

const links = [
  ["/admin/categories", "分类管理", "维护层级领域、排序、状态与团队负责人", FolderTree],
  ["/admin/teams", "团队管理", "文档维护团队、负责人、成员，团队负责领域文档结构", UsersRound],
  ["/admin/modules", "文档源 / 模块", "接入 Git 仓库、绑定能力域、版本与同步", Boxes],
  ["/admin/users", "用户管理", "管理用户、用户组、角色和状态", Users],
  ["/admin/settings", "模型设置", "对接大模型与向量化服务，启用 AI 问答", Settings],
  ["/admin/releases", "发布记录", "追踪 Pipeline、构建与发布结果", History],
  ["/admin/analytics", "阅读统计", "查看 PV、UV 和热门文档", BarChart3],
  ["/admin/search-logs", "搜索日志", "分析查询词、命中数和点击", Search],
  ["/admin/mcp-logs", "MCP 日志", "查看 AI 工具读取文档记录", MessageSquareText],
] as const;

export default function AdminPage() {
  return (
    <AdminShell title="治理控制台" kicker="Modex Admin" description="管理能力域、团队、用户、文档源与发布。">
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
