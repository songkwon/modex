"use client";

import { useEffect, useMemo, useState } from "react";
import { Check, ChevronRight, Search, X } from "lucide-react";
import { getUsers } from "@/lib/api";
import type { User } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

/**
 * Reusable multi-select for picking users (by username). Supports keyword search
 * across name/username/email/department and groups results by department.
 * Designed to scale to large user directories — searching narrows the list and
 * auto-expands matching groups.
 */
export function UserSelect({
  value,
  onChange,
  placeholder,
  single = false,
}: {
  value: string[];
  onChange: (next: string[]) => void;
  placeholder?: string;
  single?: boolean;
}) {
  const { t } = useI18n();
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
      const dept = (u.department || "").trim() || t("component.ui.userSelect.others");
      (m[dept] ||= []).push(u);
    }
    return Object.entries(m).sort((a, b) => a[0].localeCompare(b[0], "zh"));
  }, [filtered]);

  const selected = new Set(value);
  const toggle = (username: string) => {
    if (single) {
      onChange(selected.has(username) ? [] : [username]);
      return;
    }
    onChange(selected.has(username) ? value.filter((v) => v !== username) : [...value, username]);
  };

  return (
    <div className="user-select">
      {value.length > 0 ? (
        <div className="user-select__chips">
          {value.map((uname) => {
            const u = byUsername[uname];
            return (
              <span className="user-select__chip" key={uname}>
                {u?.display_name || uname}
                <button type="button" onClick={() => toggle(uname)} aria-label={t("component.ui.combobox.remove")}>
                  <X size={12} />
                </button>
              </span>
            );
          })}
        </div>
      ) : null}

      <div className="user-select__search">
        <Search size={15} />
        <input value={keyword} placeholder={placeholder ?? t("component.ui.userSelect.search_name_username_email_department")} onChange={(e) => setKeyword(e.target.value)} />
      </div>

      <div className="user-select__list">
        {groups.length === 0 ? (
          <p className="muted" style={{ fontSize: 13, padding: "8px 4px" }}>{t("component.ui.userSelect.no_matching_users")}</p>
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
