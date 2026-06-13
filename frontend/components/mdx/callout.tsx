import { ReactNode } from "react";
import { Info, Lightbulb, TriangleAlert, CircleCheck, Pencil } from "lucide-react";

type CalloutKind = "note" | "info" | "tip" | "warning" | "check" | "danger";

const KINDS: Record<CalloutKind, { icon: typeof Info }> = {
  note: { icon: Pencil },
  info: { icon: Info },
  tip: { icon: Lightbulb },
  warning: { icon: TriangleAlert },
  danger: { icon: TriangleAlert },
  check: { icon: CircleCheck }
};

function Callout({ kind, children }: { kind: CalloutKind; children: ReactNode }) {
  const Cmp = KINDS[kind].icon;
  return (
    <div className={`mdx-callout mdx-callout--${kind}`} role="note">
      <div className="mdx-callout__icon">
        <Cmp size={18} aria-hidden />
      </div>
      <div className="mdx-callout__body">{children}</div>
    </div>
  );
}

export const Note = ({ children }: { children: ReactNode }) => <Callout kind="note">{children}</Callout>;
export const Info_ = ({ children }: { children: ReactNode }) => <Callout kind="info">{children}</Callout>;
export const Tip = ({ children }: { children: ReactNode }) => <Callout kind="tip">{children}</Callout>;
export const Warning = ({ children }: { children: ReactNode }) => <Callout kind="warning">{children}</Callout>;
export const Danger = ({ children }: { children: ReactNode }) => <Callout kind="danger">{children}</Callout>;
export const Check = ({ children }: { children: ReactNode }) => <Callout kind="check">{children}</Callout>;

// Generic <Callout type="info"> form used by some Mintlify docs.
export function CalloutTag({ type = "note", children }: { type?: string; children: ReactNode }) {
  const kind = (["note", "info", "tip", "warning", "check", "danger"].includes(type) ? type : "note") as CalloutKind;
  return <Callout kind={kind}>{children}</Callout>;
}
