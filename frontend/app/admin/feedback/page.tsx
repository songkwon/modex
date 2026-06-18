"use client";

import { useState } from "react";
import { MessageCircleQuestion, Search, ThumbsDown, ThumbsUp } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { usePaged } from "@/lib/use-paged";

const PAGE_SIZE = 12;

type FeedbackLog = {
  id: string;
  doc_id: string;
  module_key: string;
  title: string;
  rating: "good" | "bad";
  comment: string;
  user_id: string;
  display_name: string;
  session_id: string;
  created_at: string;
};

export default function FeedbackPage() {
  const [keyword, setKeyword] = useState("");
  const { items: pageRows, total, page, setPage, error } = usePaged<FeedbackLog>("/api/admin/analytics/feedback", PAGE_SIZE, keyword.trim());

  return (
    <AdminShell title="文档反馈" kicker="Feedback" description="查看读者在文档底部提交的有帮助 / 需改进反馈。">
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>加载失败：{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder="搜索文档 / 模块 / 用户 / 反馈"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {total === 0 ? (
          <EmptyState
            icon={MessageCircleQuestion}
            title={keyword ? "没有匹配的反馈" : "暂无文档反馈"}
            hint={keyword ? "换个关键词试试。" : "用户点击文档底部反馈按钮后，记录会显示在这里。"}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>文档</th>
                <th>反馈</th>
                <th>内容</th>
                <th>模块</th>
                <th>用户</th>
                <th>时间</th>
              </tr>
            </thead>
            <tbody>
              {pageRows.map((log) => (
                <tr key={log.id}>
                  <td>
                    <div className="font-medium">{log.title || log.doc_id}</div>
                    <div className="muted text-xs">{log.doc_id}</div>
                  </td>
                  <td>
                    <span className={`badge ${log.rating === "good" ? "badge-success" : "badge-warn"}`}>
                      {log.rating === "good" ? <ThumbsUp size={13} /> : <ThumbsDown size={13} />}
                      {log.rating === "good" ? "有帮助" : "需改进"}
                    </span>
                  </td>
                  <td className="muted text-sm">{log.comment || "-"}</td>
                  <td>{log.module_key || "-"}</td>
                  <td>
                    {log.display_name ? (
                      <span>{log.display_name}</span>
                    ) : log.user_id ? (
                      <span className="muted">{log.user_id}</span>
                    ) : (
                      <span className="muted">匿名</span>
                    )}
                  </td>
                  <td>{log.created_at?.slice(0, 19).replace("T", " ")}</td>
                </tr>
              ))}
            </tbody>
          </table></div>
        )}
        <Pagination page={page} pageSize={PAGE_SIZE} total={total} onPage={setPage} />
      </div>
    </AdminShell>
  );
}
