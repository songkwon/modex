"use client";

import { useState } from "react";
import type { ReactNode } from "react";
import { Info } from "lucide-react";
import { Modal } from "@/components/ui/modal";
import { useI18n } from "@/lib/i18n";
import type { Category } from "@/types/modex";

export function CategoryInfoButton({
  category,
  modulesCount,
  compact = false,
}: {
  category: Category;
  modulesCount?: number;
  compact?: boolean;
}) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const team = category.responsible_team_info;
  const leaders = team?.leaders?.filter(Boolean) || [];

  return (
    <>
      <button
        type="button"
        className={compact ? "category-info-btn category-info-btn--compact" : "category-info-btn"}
        aria-label={t("component.categoryInfo.open_category_info")}
        title={t("component.categoryInfo.open_category_info")}
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          setOpen(true);
        }}
      >
        <Info size={compact ? 14 : 16} />
      </button>
      <Modal
        open={open}
        onClose={() => setOpen(false)}
        title={category.name}
        subtitle={t("component.categoryInfo.category_information")}
        width={560}
      >
        <div className="category-info-panel">
          <InfoRow label={t("component.categoryInfo.description")}>
            {category.description || t("component.categoryInfo.no_description")}
          </InfoRow>
          {typeof modulesCount === "number" ? (
            <InfoRow label={t("component.categoryInfo.document_collections")}>
              {modulesCount}
            </InfoRow>
          ) : null}
          <InfoRow label={t("component.categoryInfo.responsible_team")}>
            {team ? (
              <div className="category-info-stack">
                <strong>{team.name || team.key}</strong>
                {team.description ? <span>{team.description}</span> : null}
              </div>
            ) : category.responsible_team ? (
              <span>{category.responsible_team}</span>
            ) : (
              <span className="muted">{t("component.categoryInfo.not_assigned")}</span>
            )}
          </InfoRow>
          <InfoRow label={t("component.categoryInfo.owners")}>
            {leaders.length > 0 ? (
              <div className="category-info-tags">
                {leaders.map((leader) => <span className="badge badge-success" key={leader}>{leader}</span>)}
              </div>
            ) : (
              <span className="muted">{t("component.categoryInfo.not_specified")}</span>
            )}
          </InfoRow>
          <p className="category-info-note">
            {t("component.categoryInfo.sync_contact_hint")}
          </p>
        </div>
      </Modal>
    </>
  );
}

function InfoRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="category-info-row">
      <div className="category-info-row-label">{label}</div>
      <div className="category-info-row-value">{children}</div>
    </div>
  );
}
