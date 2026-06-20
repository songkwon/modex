"use client";

import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { List } from "lucide-react";
import { useI18n } from "@/lib/i18n";

type Item = { id: string; text: string; level: number };
type Point = { x: number; y: number };
type ActiveRange = { top: number; height: number; dotX: number; dotY: number };

// Fumadocs-style "On this page" outline.
// - Headings visible in the current viewport are highlighted.
// - A theme-coloured polyline connects every TOC item, with rounded corners at
//   depth changes.
// - The visible range is painted in full accent colour over the rail via a
//   clipPath so the active segment follows the exact line shape.
// - A small dot marks the end of the active range (the lowest visible heading).
export function DocToc() {
  const { t } = useI18n();
  const [items, setItems] = useState<Item[]>([]);
  const [activeIds, setActiveIds] = useState<Set<string>>(new Set());
  const [activeRange, setActiveRange] = useState<ActiveRange | null>(null);
  const [rail, setRail] = useState<{ path: string } | null>(null);
  const clipId = useId().replace(/:/g, "");
  const navRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<Map<string, HTMLAnchorElement>>(new Map());
  const clickLockRef = useRef<string | null>(null);
  const clickTimeoutRef = useRef<number | null>(null);

  // 1. Read headings from the rendered document body.
  useEffect(() => {
    const refresh = () => {
      const heads = Array.from(
        document.querySelectorAll<HTMLElement>(".doc-page .mdx h1, .doc-page .mdx h2, .doc-page .mdx h3")
      ).filter((h) => h.id);
      setItems(heads.map((h) => ({ id: h.id, text: h.textContent || "", level: Number(h.tagName[1]) })));
    };
    refresh();
    const root = document.querySelector(".doc-page");
    if (!root) return;
    const observer = new MutationObserver(refresh);
    observer.observe(root, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, []);

  // 2. Track headings visible in the current viewport.
  useEffect(() => {
    if (items.length === 0) return;
    const heads = currentHeadingElements(items);
    let frame = 0;

    const update = () => {
      if (frame) window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(() => {
        if (clickLockRef.current) {
          setActiveIds(new Set([clickLockRef.current]));
          return;
        }
        setActiveIds(new Set(activeHeadingsInViewport(heads)));
      });
    };

    update();
    window.addEventListener("scroll", update, { passive: true });
    window.addEventListener("resize", update);
    return () => {
      if (frame) window.cancelAnimationFrame(frame);
      window.removeEventListener("scroll", update);
      window.removeEventListener("resize", update);
    };
  }, [items]);

  const lineX = (level: number) => {
    const minLevel = items.length ? Math.min(...items.map((i) => i.level)) : 1;
    return 8 + (level - minLevel) * 14;
  };

  // 3. Measure TOC item positions and build the connecting rail + thumb.
  useLayoutEffect(() => {
    if (!navRef.current || items.length === 0) return;
    const navRect = navRef.current.getBoundingClientRect();

    const positions: Point[] = [];
    items.forEach((it) => {
      const el = itemRefs.current.get(it.id);
      if (!el) return;
      const rect = el.getBoundingClientRect();
      positions.push({ x: lineX(it.level), y: rect.top + rect.height / 2 - navRect.top });
    });
    if (positions.length === 0) return;

    setRail({
      path: buildRailPath(positions),
    });

    if (activeIds.size === 0) {
      setActiveRange(null);
      return;
    }
    const itemById = new Map(items.map((it) => [it.id, it]));
    let minY = Infinity;
    let maxY = -Infinity;
    let dotX = 0;
    activeIds.forEach((id) => {
      const el = itemRefs.current.get(id);
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const centerY = rect.top + rect.height / 2 - navRect.top;
      if (centerY > maxY) {
        maxY = centerY;
        const it = itemById.get(id);
        dotX = it ? lineX(it.level) : 0;
      }
      minY = Math.min(minY, centerY);
    });
    const top = Math.max(0, minY - 6);
    const height = Math.max(12, maxY - minY + 12);
    setActiveRange({
      top,
      height,
      dotX,
      dotY: top + height,
    });
  }, [items, activeIds]);

  if (items.length === 0) return null;

  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>, id: string) => {
    e.preventDefault();
    const el = document.getElementById(id);
    if (!el) return;

    clickLockRef.current = id;
    setActiveIds(new Set([id]));
    if (clickTimeoutRef.current) window.clearTimeout(clickTimeoutRef.current);
    clickTimeoutRef.current = window.setTimeout(() => {
      clickLockRef.current = null;
    }, 900);

    el.scrollIntoView({ behavior: "smooth", block: "start" });
    history.replaceState(null, "", `#${id}`);
  };

  return (
    <>
      <p className="doc-toc-title doc-toc-head">
        <List size={14} /> {t("legacy.a610e0c62b3e")}
      </p>
      <div className="doc-toc-wrap" ref={navRef}>
        {rail ? (
          <svg
            className="doc-toc-rail"
            aria-hidden="true"
            preserveAspectRatio="none"
          >
            <defs>
              <clipPath id={clipId}>
                <rect
                  x="-10"
                  y={activeRange?.top ?? 0}
                  width="300"
                  height={activeRange?.height ?? 0}
                />
              </clipPath>
            </defs>
            <path className="doc-toc-rail__base" d={rail.path} />
            {activeRange ? (
              <>
                <path className="doc-toc-rail__active" d={rail.path} clipPath={`url(#${clipId})`} />
                <circle className="doc-toc-rail__end-dot" cx={activeRange.dotX} cy={activeRange.dotY} r="2.5" />
              </>
            ) : null}
          </svg>
        ) : null}
        <nav className="doc-toc-tree" aria-label={t("legacy.a610e0c62b3e")}>
          {items.map((it) => (
            <a
              key={it.id}
              href={`#${it.id}`}
              ref={(el) => {
                if (el) itemRefs.current.set(it.id, el);
              }}
              className={`doc-toc-link${activeIds.has(it.id) ? " active" : ""}`}
              style={{ paddingLeft: `${lineX(it.level) + 14}px` }}
              onClick={(e) => handleClick(e, it.id)}
            >
              {it.text}
            </a>
          ))}
        </nav>
      </div>
    </>
  );
}

function activeHeadingsInViewport(heads: HTMLElement[]) {
  if (heads.length === 0) return [];
  const top = 0;
  const bottom = window.innerHeight;
  const visible = heads
    .filter((h) => {
      const rect = h.getBoundingClientRect();
      return rect.bottom >= top && rect.top <= bottom;
    })
    .map((h) => h.id);
  if (visible.length > 0) return visible;

  let current = heads[0]?.id ? [heads[0].id] : [];
  for (const h of heads) {
    const rect = h.getBoundingClientRect();
    if (rect.top > top) {
      break;
    }
    current = [h.id];
  }
  return current;
}

function currentHeadingElements(items: Item[]) {
  const all = Array.from(document.querySelectorAll<HTMLElement>(".doc-page .mdx h1, .doc-page .mdx h2, .doc-page .mdx h3"));
  const byId = new Map(all.filter((h) => h.id).map((h) => [h.id, h]));
  return items.map((it) => byId.get(it.id)).filter(Boolean) as HTMLElement[];
}

function buildRailPath(points: Point[]): string {
  if (points.length === 0) return "";
  const radius = 6;
  let d = `M ${points[0].x} ${points[0].y}`;
  for (let i = 1; i < points.length; i++) {
    const prev = points[i - 1];
    const cur = points[i];
    if (prev.x === cur.x) {
      d += ` L ${cur.x} ${cur.y}`;
      continue;
    }
    // Draw a rounded corner between two horizontal offsets.
    const midY = prev.y + (cur.y - prev.y) / 2;
    const r = Math.min(radius, Math.abs(cur.y - prev.y) / 2);
    const dir = cur.x > prev.x ? 1 : -1;
    d += ` L ${prev.x} ${midY - r}`;
    d += ` Q ${prev.x} ${midY}, ${prev.x + dir * r} ${midY}`;
    d += ` L ${cur.x - dir * r} ${midY}`;
    d += ` Q ${cur.x} ${midY}, ${cur.x} ${midY + r}`;
    d += ` L ${cur.x} ${cur.y}`;
  }
  return d;
}
