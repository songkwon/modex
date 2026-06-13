"use client";

import { useEffect, useMemo, useState } from "react";
import { Check, ChevronRight, Search, X } from "lucide-react";
import { getUsers } from "@/lib/api";
import type { User } from "@/types/modex";

/**
 * Reusable multi-select for picking users (by username). Supports keyword search
 * across name/username/email/department and groups results by department.
 * Designed to scale to large user directories — searching narrows the list and
 * auto-expands matching groups.
 */
export function UserSelect({
  value,
  onChange,
  placeholder = "搜索姓名 / 用户名 / 邮箱 / 部门",
}: {
  value: string[];
  onChange: (next: string[]) => void;
  placeholder?: string;
}) {
  const [users, setUsers] = useState<User[]>([]);
  const [keyword, setKeyword] = useState("");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  useEffect(() => {
    let cancelled = false;
    getUsers()
      .then((u) => { if (!cancelled) setUsers(u || []); })
      .catch(() => {});
    return () => { cancelled = true; };
  }, []);

  const byUsername = useMemo(() => {
    const m: Record<string, User> = {};
    for (const u of users) m[u.username] = u;
    return m;
  }, [users]);

  const q = keyword.trim().toLowerCase();
  const filtered = useMemo(
    () =>
      users.filter(
        (u) => !q || [u.username, u.display_name, u.email, u.department].some((f) => (f || "").toLowerCase().includes(q)),
      ),
    [users, q],
  );

  const groups = useMemo(() => {
    const m: Record<string, User[]> = {};
    for (const u of filtered) {
      const dept = (u.department || "").trim() || "其他";
      (m[dept] ||= []).push(u);
    }
    return Object.entries(m).sort((a, b) => a[0].localeCompare(b[0], "zh"));
  }, [filtered]);

  const selected = new Set(value);
  const toggle = (username: string) =>
    onChange(selected.has(username) ? value.filter((v) => v !== username) : [...value, username]);

  return (
    <div className="user-select">
      {value.length > 0 ? (
        <div className="user-select__chips">
          {value.map((uname) => {
            const u = byUsername[uname];
            return (
              <span className="user-select__chip" key={uname}>
                {u?.display_name || uname}
                <button type="button" onClick={() => toggle(uname)} aria-label="移除">
                  <X size={12} />
                </button>
              </span>
            );
          })}
        </div>
      ) : null}

      <div className="user-select__search">
        <Search size={15} />
        <input value={keyword} placeholder={placeholder} onChange={(e) => setKeyword(e.target.value)} />
      </div>

      <div className="user-select__list">
        {groups.length === 0 ? (
          <p className="muted" style={{ fontSize: 13, padding: "8px 4px" }}>没有匹配的用户</p>
        ) : (
          groups.map(([dept, members]) => {
            const isCollapsed = q ? false : collapsed[dept];
            return (
              <div className="user-select__group" key={dept}>
                <button
                  type="button"
                  className="user-select__group-head"
                  onClick={() => setCollapsed((c) => ({ ...c, [dept]: !c[dept] }))}
                >
                  <ChevronRight size={14} className={`user-select__chevron${isCollapsed ? "" : " is-open"}`} />
                  <span>{dept}</span>
                  <span className="user-select__count">{members.length}</span>
                </button>
                {!isCollapsed ? (
                  <div className="user-select__members">
                    {members.map((u) => {
                      const on = selected.has(u.username);
                      return (
                        <button type="button" key={u.id} className={`user-select__item${on ? " is-selected" : ""}`} onClick={() => toggle(u.username)}>
                          <span className="user-select__check">{on ? <Check size={13} /> : null}</span>
                          <span className="user-select__name">{u.display_name || u.username}</span>
                          <span className="user-select__sub">{u.username}{u.email ? ` · ${u.email}` : ""}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
