"use client";

import type { KeyboardEvent } from "react";
import { ArrowRight, BookOpen, Box, Boxes, Cpu, Layers, Layout, Package, Wrench } from "lucide-react";
import type { Category, ModuleInfo } from "@/types/modex";
import { useI18n } from "@/lib/i18n";
import { CategoryInfoButton } from "@/components/category-info-button";

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

function categoryIDs(category: Category): string[] {
  return [category.id, ...(category.children || []).flatMap(categoryIDs)];
}

function countModules(modules: ModuleInfo[], category: Category) {
  const ids = new Set(categoryIDs(category));
  return modules.filter((m) => (m.category_ids || []).some((id) => ids.has(id))).length;
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
  const { t } = useI18n();
  function activateCard(event: KeyboardEvent, category: Category) {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onSelect(category);
    }
  }
  return (
    <div className="platform-grid">
      {categories.map((cat) => {
        const Icon = iconFor(cat);
        const total = countModules(modules, cat);
        const children = cat.children || [];
        return (
          <article
            className="platform-card"
            key={cat.id}
            role="button"
            tabIndex={0}
            onClick={() => onSelect(cat)}
            onKeyDown={(event) => activateCard(event, cat)}
          >
            <div className="platform-card-info">
              <CategoryInfoButton category={cat} modulesCount={total} compact />
            </div>
            <div className="platform-card-head">
              <span className="platform-icon"><Icon size={20} /></span>
              <div className="platform-titles">
                <div className="platform-name">{cat.name}</div>
                <div className="platform-count">{total} {t("component.platformCards.document_collections")}</div>
              </div>
            </div>
            {cat.description ? <p className="platform-desc muted">{cat.description}</p> : null}
            {children.length > 0 ? (
              <div className="platform-subs">
                {children.map((sub) => {
                  const subTotal = countModules(modules, sub);
                  return (
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
                      <em className="platform-sub-count">{subTotal}</em>
                      <CategoryInfoButton category={sub} modulesCount={subTotal} compact />
                    </span>
                  );
                })}
              </div>
            ) : null}
            <span className="platform-cta">{t("component.platformCards.browse_documentation")} <ArrowRight size={14} /></span>
          </article>
        );
      })}
    </div>
  );
}
