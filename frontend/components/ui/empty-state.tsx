import type { LucideIcon } from "lucide-react";
import { Inbox } from "lucide-react";

// EmptyState is the single, shared "no data yet" placeholder used by every admin
// table so all缺省页 look identical. Pass an icon, a title, and a short hint;
// optionally an action node (e.g. a "新增" button).
export function EmptyState({
  icon: Icon = Inbox,
  title,
  hint,
  action,
}: {
  icon?: LucideIcon;
  title: string;
  hint?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="empty-state-box">
      <span className="empty-state-icon">
        <Icon size={24} />
      </span>
      <div className="empty-state-title">{title}</div>
      {hint ? <p className="empty-state-text">{hint}</p> : null}
      {action}
    </div>
  );
}
