"use client";

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown, X } from "lucide-react";

export type ComboOption = { value: string; label: string; hint?: string; depth?: number };

/**
 * Searchable select with chips. Supports multi-select (default) and single
 * select. The dropdown is rendered in a portal positioned under the control so
 * it is never clipped by a scrolling modal body. `allowCreate` lets the user
 * add free-form values not in `options`.
 */
export function Combobox({
  options,
  value,
  onChange,
  placeholder = "搜索并选择…",
  multiple = true,
  allowCreate = false,
}: {
  options: ComboOption[];
  value: string[];
  onChange: (next: string[]) => void;
  placeholder?: string;
  multiple?: boolean;
  allowCreate?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const [rect, setRect] = useState<{ left: number; top: number; width: number } | null>(null);
  const ref = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const reposition = () => {
    if (!ref.current) return;
    const r = ref.current.getBoundingClientRect();
    setRect({ left: r.left, top: r.bottom + 6, width: r.width });
  };

  useLayoutEffect(() => {
    if (open) reposition();
  }, [open, value, query]);

  useEffect(() => {
    if (!open) return;
    const onScroll = () => reposition();
    const onDoc = (e: MouseEvent) => {
      const t = e.target as Node;
      if (ref.current?.contains(t) || menuRef.current?.contains(t)) return;
      setOpen(false);
    };
    window.addEventListener("scroll", onScroll, true);
    window.addEventListener("resize", onScroll);
    document.addEventListener("mousedown", onDoc);
    return () => {
      window.removeEventListener("scroll", onScroll, true);
      window.removeEventListener("resize", onScroll);
      document.removeEventListener("mousedown", onDoc);
    };
  }, [open]);

  // Strip any leading indentation (used for tree depth in the menu) so chips stay clean.
  const labelFor = (v: string) => (options.find((o) => o.value === v)?.label || v).trimStart();

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return options.filter((o) => !q || o.label.toLowerCase().includes(q) || o.value.toLowerCase().includes(q));
  }, [options, query]);

  const canCreate =
    allowCreate &&
    query.trim() &&
    !options.some((o) => o.value.toLowerCase() === query.trim().toLowerCase()) &&
    !value.includes(query.trim());

  function toggle(v: string) {
    if (multiple) {
      onChange(value.includes(v) ? value.filter((x) => x !== v) : [...value, v]);
    } else {
      onChange([v]);
      setOpen(false);
    }
    setQuery("");
  }

  function commitActive() {
    if (active < filtered.length) toggle(filtered[active].value);
    else if (canCreate) toggle(query.trim());
  }

  const visibleChips = value.filter(Boolean);

  const menu =
    open && rect && typeof document !== "undefined"
      ? createPortal(
          <div
            ref={menuRef}
            className="combo-menu"
            style={{ position: "fixed", left: rect.left, top: rect.top, width: rect.width }}
          >
            {filtered.map((o, i) => {
              const selected = value.includes(o.value);
              return (
                <button
                  type="button"
                  key={o.value}
                  className={`combo-option${selected ? " selected" : ""}${i === active ? " active" : ""}`}
                  style={o.depth ? { paddingLeft: 10 + o.depth * 16 } : undefined}
                  onMouseEnter={() => setActive(i)}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => toggle(o.value)}
                >
                  <span>{o.label}</span>
                  {o.hint ? <span className="tree-key">{o.hint}</span> : null}
                  {selected ? <Check size={15} className="combo-check" /> : null}
                </button>
              );
            })}
            {canCreate ? (
              <button
                type="button"
                className={`combo-option${active >= filtered.length ? " active" : ""}`}
                onMouseEnter={() => setActive(filtered.length)}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => toggle(query.trim())}
              >
                新增 “{query.trim()}”
              </button>
            ) : null}
            {!filtered.length && !canCreate ? <div className="combo-empty">无匹配项</div> : null}
          </div>,
          document.body,
        )
      : null;

  return (
    <div className={`combobox${open ? " open" : ""}`} ref={ref}>
      <div className="combo-control" onClick={() => setOpen(true)}>
        {visibleChips.map((v) => (
          <span className="combo-chip" key={v}>
            {labelFor(v)}
            <button
              type="button"
              aria-label="移除"
              onClick={(e) => {
                e.stopPropagation();
                onChange(value.filter((x) => x !== v));
              }}
            >
              <X size={13} />
            </button>
          </span>
        ))}
        <input
          value={query}
          placeholder={visibleChips.length ? "" : placeholder}
          onChange={(e) => {
            setQuery(e.target.value);
            setOpen(true);
            setActive(0);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={(e) => {
            if (e.key === "ArrowDown") { e.preventDefault(); setActive((a) => Math.min(a + 1, filtered.length)); }
            else if (e.key === "ArrowUp") { e.preventDefault(); setActive((a) => Math.max(a - 1, 0)); }
            else if (e.key === "Enter") { e.preventDefault(); commitActive(); }
            else if (e.key === "Escape") { setOpen(false); }
            else if (e.key === "Backspace" && !query && value.length) onChange(value.slice(0, -1));
          }}
        />
        <ChevronDown size={15} style={{ marginLeft: "auto", color: "hsl(var(--muted))", flex: "none" }} />
      </div>
      {menu}
    </div>
  );
}
