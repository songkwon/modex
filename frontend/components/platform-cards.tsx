"use client";

import { ArrowRight, BookOpen, Box, Boxes, Cpu, Layers, Layout, Package, Wrench } from "lucide-react";
import type { Category, ModuleInfo } from "@/types/modex";

const ICONS: Record<string, typeof Boxes> = {
  wrench: Wrench,
  package: Package,
  box: Box,
  layers: Layers,
  layout: Layout,
  "book-open": BookOpen,
  book: BookOpen,
  cpu: Cpu
};

function iconFor(cat: Category) {
  return ICONS[cat.icon || ""] || Boxes;
}

// countModules returns how many modules belong to a category id. Modules carry
// the full category path (parent + child ids), so a simple membership test
// counts both a platform and its sub-platforms correctly.
function countModules(modules: ModuleInfo[], categoryId: string) {
  return modules.filter((m) => (m.category_ids || []).includes(categoryId)).length;
}

export function PlatformCards({
  categories,
  modules,
  onSelect
}: {
  categories: Category[];
  modules: ModuleInfo[];
  onSelect: (category: Category) => void;
}) {
  return (
    <div className="platform-grid">
      {categories.map((cat) => {
        const Icon = iconFor(cat);
        const total = countModules(modules, cat.id);
        const children = cat.children || [];
        return (
          <button className="platform-card" key={cat.id} onClick={() => onSelect(cat)}>
            <div className="platform-card-head">
              <span className="platform-icon"><Icon size={20} /></span>
              <div className="platform-titles">
                <div className="platform-name">{cat.name}</div>
                <div className="platform-count">{total} 个文档集合</div>
              </div>
            </div>
            {cat.description ? <p className="platform-desc muted">{cat.description}</p> : null}
            {children.length > 0 ? (
              <div className="platform-subs">
                {children.map((sub) => (
                  <span
                    className="platform-sub"
                    key={sub.id}
                    role="link"
                    tabIndex={0}
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelect(sub);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        e.stopPropagation();
                        onSelect(sub);
                      }
                    }}
                  >
                    {sub.name}
                    <em className="platform-sub-count">{countModules(modules, sub.id)}</em>
                  </span>
                ))}
              </div>
            ) : null}
            <span className="platform-cta">浏览文档 <ArrowRight size={14} /></span>
          </button>
        );
      })}
    </div>
  );
}
