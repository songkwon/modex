"use client";

import { useState } from "react";
import { Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { usePaged } from "@/lib/use-paged";

const PAGE_SIZE = 12;

type SearchLog = {
  id: string;
  user_id: string;
  display_name: string;
  ip_address: string;
  query: string;
  mode: string;
  result_count: number;
  clicked_doc_id: string;
  searched_at: string;
};

export default function SearchLogsPage() {
  const [keyword, setKeyword] = useState("");
  const { items: pageRows, total, page, setPage, error } = usePaged<SearchLog>("/api/admin/analytics/search", PAGE_SIZE, keyword.trim());

  return (
    <AdminShell title="搜索日志" kicker="Search Analytics" description="观察高频搜索词、无结果查询和搜索点击行为。">
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>加载失败：{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder="搜索查询词 / 用户 / IP"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {total === 0 ? (
          <EmptyState
            icon={Search}
            title={keyword ? "没有匹配的搜索日志" : "暂无搜索日志"}
            hint={keyword ? "换个关键词试试。" : "用户执行一次搜索（按回车或点击结果）后，记录会显示在这里。"}
          />
        ) : (
          <>
            <div className="table-scroll"><table className="data-table">
              <thead>
                <tr>
                  <th>查询词</th>
                  <th>模式</th>
                  <th>结果数</th>
                  <th>用户</th>
                  <th>点击</th>
                  <th>时间</th>
                </tr>
              </thead>
              <tbody>
                {pageRows.map((log) => (
                  <tr key={log.id}>
                    <td className="font-medium">{log.query || "-"}</td>
                    <td><span className="tag">{log.mode}</span></td>
                    <td>{log.result_count}</td>
                    <td>
                      {log.display_name ? (
                        <span>{log.display_name}</span>
                      ) : log.user_id ? (
                        <span className="muted">{log.user_id}</span>
                      ) : log.ip_address ? (
                        <span className="muted">{log.ip_address}</span>
                      ) : (
                        <span className="muted">-</span>
                      )}
                    </td>
                    <td>{log.clicked_doc_id || "-"}</td>
                    <td>{log.searched_at?.slice(0, 19).replace("T", " ")}</td>
                  </tr>
                ))}
              </tbody>
            </table></div>
            <Pagination page={page} pageSize={PAGE_SIZE} total={total} onPage={setPage} />
          </>
        )}
      </div>
    </AdminShell>
  );
}
