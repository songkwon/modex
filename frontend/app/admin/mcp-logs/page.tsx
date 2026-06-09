import { AdminShell } from "@/components/admin-shell";
import { api } from "@/lib/api";

type MCPLog = {
  id: string;
  tool_name: string;
  user_id: string;
  query: string;
  input_json: string;
  result_count: number;
  created_at: string;
};

export default async function MCPLogsPage() {
  const data = await api<MCPLog[]>("/api/admin/analytics/mcp");
  return (
    <AdminShell title="MCP 日志" kicker="AI Access" description="记录 AI 工具通过 MCP 读取模块、版本、搜索结果和文档页面的行为。">
      <section className="panel">
        {data.length === 0 ? (
          <div className="empty-state">
            <div>
              <div className="font-semibold text-slate-700">暂无 MCP 调用</div>
              <p className="mt-2 text-sm">启动 docs-mcp-server 并调用工具后，这里会出现记录。</p>
            </div>
          </div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>工具</th>
                <th>查询</th>
                <th>结果数</th>
                <th>用户</th>
                <th>输入</th>
                <th>时间</th>
              </tr>
            </thead>
            <tbody>
              {data.map((log) => (
                <tr key={log.id}>
                  <td><span className="tag">{log.tool_name}</span></td>
                  <td className="font-medium">{log.query || "-"}</td>
                  <td>{log.result_count}</td>
                  <td>{log.user_id || "-"}</td>
                  <td><span className="code-chip">{log.input_json || "{}"}</span></td>
                  <td>{log.created_at?.slice(0, 19).replace("T", " ")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </AdminShell>
  );
}
