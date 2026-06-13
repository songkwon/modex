"use client";

import { useEffect, useMemo, useState } from "react";
import { MessageSquareText, Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { api } from "@/lib/api";

const PAGE_SIZE = 12;

type MCPLog = {
  id: string;
  tool_name: string;
  display_name: string;
  user_id: string;
  query: string;
  input_json: string;
  result_count: number;
  created_at: string;
};

export default function MCPLogsPage() {
  const [data, setData] = useState<MCPLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const logs = await api<MCPLog[]>("/api/admin/analytics/mcp");
        if (!cancelled) setData(logs || []);
      } catch (e) {
        if (!cancelled) setError(String(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const filtered = useMemo(() => {
    const q = keyword.trim().toLowerCase();
    if (!q) return data;
    return data.filter((l) =>
      (l.tool_name || "").toLowerCase().includes(q) ||
      (l.query || "").toLowerCase().includes(q) ||
      (l.display_name || "").toLowerCase().includes(q) ||
      (l.user_id || "").toLowerCase().includes(q),
    );
  }, [data, keyword]);
  const pageRows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  return (
    <AdminShell title="MCP 日志" kicker="AI Access" description="记录 AI 工具通过 MCP 读取模块、版本、搜索结果与文档页面的行为。">
      {error ? (
        <div className="panel badge-danger" style={{ borderRadius: 12 }}>
          加载失败：{error}（可能需要超级管理员权限）
        </div>
      ) : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder="搜索工具 / 查询 / 用户"
            value={keyword}
            onChange={(e) => { setKeyword(e.target.value); setPage(1); }}
          />
        </div>
      </div>

      <div className="table-card">
        {loading ? (
          <div className="muted text-sm" style={{ padding: 18 }}>加载中...</div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={MessageSquareText}
            title={keyword ? "没有匹配的记录" : "暂无 MCP 调用"}
            hint={keyword ? "换个关键词试试。" : "配置并连接 MCP 服务后，AI 工具读取文档的记录会出现在这里。"}
          />
        ) : (
          <>
            <div className="table-scroll"><table className="data-table">
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
                {pageRows.map((log) => (
                  <tr key={log.id}>
                    <td><span className="tag">{log.tool_name}</span></td>
                    <td className="font-medium">{log.query || "-"}</td>
                    <td>{log.result_count}</td>
                    <td>
                      {log.display_name ? (
                        <span>{log.display_name}</span>
                      ) : log.user_id ? (
                        <span className="muted">{log.user_id}</span>
                      ) : (
                        <span className="muted">-</span>
                      )}
                    </td>
                    <td><span className="code-chip">{log.input_json || "{}"}</span></td>
                    <td>{log.created_at?.slice(0, 19).replace("T", " ")}</td>
                  </tr>
                ))}
              </tbody>
            </table></div>
            <Pagination page={page} pageSize={PAGE_SIZE} total={filtered.length} onPage={setPage} />
          </>
        )}
      </div>
    </AdminShell>
  );
}
