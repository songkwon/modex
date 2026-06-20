"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { ChevronRight, FolderTree, GripVertical, Pencil, Plus, Trash2 } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import { EmptyState } from "@/components/ui/empty-state";
import { CategoryIcon, IconPicker } from "@/components/ui/icon-picker";
import { getManagedCategories, getMe, getTeams, createCategory, updateCategory, deleteCategory, moveCategory } from "@/lib/api";
import type { Category, Team } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

type FlatRow = { category: Category; depth: number };
type DropPos = "before" | "after" | "into";
type DropTarget = { id: string; pos: DropPos };

function flatten(categories: Category[] | null | undefined, depth = 0): FlatRow[] {
  if (!categories) return [];
  return categories.flatMap((c) => [{ category: c, depth }, ...flatten(c.children || [], depth + 1)]);
}

// Is `maybeAncestor` an ancestor of (or equal to) `node`? Prevents dropping a
// node into its own subtree.
function isAncestor(maybeAncestor: Category, node: Category): boolean {
  if (maybeAncestor.id === node.id) return true;
  return (maybeAncestor.children || []).some((c) => isAncestor(c, node));
}

function TreeNode({
  category,
  expanded,
  toggle,
  onAddSub,
  onEdit,
  onDelete,
  drag,
}: {
  category: Category;
  expanded: Set<string>;
  toggle: (id: string) => void;
  onAddSub: (id: string) => void;
  onEdit: (c: Category) => void;
  onDelete: (id: string, name: string) => void;
  drag: {
    draggingId: string | null;
    dropTarget: DropTarget | null;
    onDragStart: (id: string) => void;
    onDragEnd: () => void;
    onDragOver: (e: React.DragEvent, id: string) => void;
    onDrop: (e: React.DragEvent, id: string) => void;
  };
}) {
  const { t } = useI18n();
  const hasChildren = !!category.children?.length;
  const open = expanded.has(category.id);
  const dt = drag.dropTarget;
  const dropCls =
    dt && dt.id === category.id ? ` drop-${dt.pos}` : "";
  const dragCls = drag.draggingId === category.id ? " dragging" : "";

  return (
    <div>
      <div
        className={`tree-row${dropCls}${dragCls}`}
        draggable
        onDragStart={() => drag.onDragStart(category.id)}
        onDragEnd={drag.onDragEnd}
        onDragOver={(e) => drag.onDragOver(e, category.id)}
        onDrop={(e) => drag.onDrop(e, category.id)}
      >
        <span className="tree-grip" title={t("admin.categories.drag_to_reorder_adjust_hierarchy")}><GripVertical size={14} /></span>
        {hasChildren ? (
          <span className={`tree-twist${open ? " open" : ""}`} onClick={() => toggle(category.id)}>
            <ChevronRight size={15} />
          </span>
        ) : (
          <span className="tree-twist" style={{ visibility: "hidden" }}><ChevronRight size={15} /></span>
        )}
        <span className="tree-icon"><CategoryIcon name={category.icon} /></span>
        <span className="tree-label">{category.name}</span>
        <span className="tree-spacer" />
        <div className="row-actions">
          <button className="icon-btn" onClick={() => onAddSub(category.id)} title={t("admin.categories.add_subcategory")}><Plus size={14} /></button>
          <button className="icon-btn" onClick={() => onEdit(category)} title={t("admin.categories.edit")}><Pencil size={14} /></button>
          <button className="icon-btn danger" onClick={() => onDelete(category.id, category.name)} title={t("admin.categories.delete")}><Trash2 size={14} /></button>
        </div>
      </div>
      {hasChildren && open ? (
        <div className="tree-children">
          {category.children!.map((child) => (
            <TreeNode key={child.id} category={child} expanded={expanded} toggle={toggle} onAddSub={onAddSub} onEdit={onEdit} onDelete={onDelete} drag={drag} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

export default function AdminCategoriesPage() {
  const { t } = useI18n();
  const [categories, setCategories] = useState<Category[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [error, setError] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [modalOpen, setModalOpen] = useState(false);
  const [data, setData] = useState<Partial<Category> & { id?: string }>({});

  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<DropTarget | null>(null);
  const [isSuper, setIsSuper] = useState(false);
  const byId = useRef<Map<string, Category>>(new Map());

  useEffect(() => {
    getMe().then((me) => setIsSuper(!!me.is_super_admin)).catch(() => {});
    // Teams are only used for responsible-team labels/selector and are
    // super-admin-only; tolerate a 403 for team admins instead of blanking the tree.
    getTeams().then((ts) => setTeams(ts || [])).catch(() => {});
  }, []);

  async function refresh() {
    try {
      const tree = await getManagedCategories();
      const safe = tree || [];
      setCategories(safe);
      byId.current = new Map(flatten(safe).map((r) => [r.category.id, r.category]));
      setExpanded((prev) => (prev.size ? prev : new Set(flatten(safe).map((r) => r.category.id))));
      setError("");
    } catch (e) {
      setError(String(e));
    }
  }

  useEffect(() => { refresh(); }, []);

  const parentOptions: ComboOption[] = useMemo(
    () => flatten(categories).map((r) => ({ value: r.category.id, label: `${"   ".repeat(r.depth)}${r.category.name}`, hint: r.category.key })),
    [categories],
  );
  const teamOptions: ComboOption[] = (teams || []).map((t) => ({ value: t.key, label: t.name || t.key, hint: t.key }));

  function toggle(id: string) {
    setExpanded((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }
  function openCreate() { setData({}); setModalOpen(true); setError(""); }
  function openAddSub(parentId: string) { setData({ parent_id: parentId }); setModalOpen(true); setError(""); }
  function openEdit(c: Category) {
    setData({ id: c.id, key: c.key, name: c.name, description: c.description, parent_id: c.parent_id, icon: c.icon, sort_order: c.sort_order, responsible_team: c.responsible_team });
    setModalOpen(true);
    setError("");
  }

  // --- drag & drop: reorder among siblings + reparent (drop "into") ---
  function onDragOver(e: React.DragEvent, id: string) {
    if (!draggingId || draggingId === id) return;
    const dragged = byId.current.get(draggingId);
    const target = byId.current.get(id);
    if (!dragged || !target || isAncestor(dragged, target)) return; // no drop into own subtree
    e.preventDefault();
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const y = e.clientY - rect.top;
    const pos: DropPos = y < rect.height * 0.28 ? "before" : y > rect.height * 0.72 ? "after" : "into";
    setDropTarget((prev) => (prev?.id === id && prev.pos === pos ? prev : { id, pos }));
  }

  async function onDrop(e: React.DragEvent, id: string) {
    e.preventDefault();
    const dt = dropTarget;
    const draggedId = draggingId;
    setDraggingId(null);
    setDropTarget(null);
    if (!draggedId || !dt || draggedId === id) return;
    const target = byId.current.get(id);
    const dragged = byId.current.get(draggedId);
    if (!target || !dragged || isAncestor(dragged, target)) return;

    let parentId: string;
    let index: number;
    if (dt.pos === "into") {
      parentId = target.id;
      index = (target.children || []).length; // append as last child
      setExpanded((prev) => new Set(prev).add(target.id));
    } else {
      parentId = target.parent_id || "";
      const siblings = (parentId ? byId.current.get(parentId)?.children : categories) || [];
      const filtered = siblings.filter((c) => c.id !== draggedId);
      const targetIdx = filtered.findIndex((c) => c.id === target.id);
      index = dt.pos === "before" ? targetIdx : targetIdx + 1;
    }
    try {
      await moveCategory(draggedId, { parent_id: parentId, index });
      await refresh();
    } catch (err) {
      setError(String(err));
    }
  }

  async function submit() {
    setError("");
    try {
      const payload: Partial<Category> = {
        key: data.key,
        name: data.name || data.key,
        description: data.description || undefined,
        parent_id: data.parent_id || undefined,
        icon: data.icon || undefined,
        sort_order: data.sort_order ? Number(data.sort_order) : 0,
        responsible_team: data.responsible_team || undefined,
      };
      if (data.id) await updateCategory(data.id, payload);
      else await createCategory(payload);
      setModalOpen(false);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  async function remove(id: string, name: string) {
    if (!confirm(t("admin.categories.delete_category_value1_subcategories_must_be_deleted_first", { value1: name }))) return;
    try {
      await deleteCategory(id);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  const drag = {
    draggingId,
    dropTarget,
    onDragStart: (id: string) => setDraggingId(id),
    onDragEnd: () => { setDraggingId(null); setDropTarget(null); },
    onDragOver,
    onDrop,
  };

  return (
    <AdminShell
      title={t("component.adminShell.category_management")}
      kicker="Categories"
      description={t("admin.categories.hierarchical_categories_support_arbitrary_nesting_drag_directly_on")}
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <div className="admin-toolbar">
        <div className="muted" style={{ fontSize: 13 }}>{flatten(categories).length} {t("admin.categories.category_nodes_drag_cards_to_reorder_or_change")}</div>
        <div className="admin-toolbar-actions">
          {isSuper ? <button className="button button-primary" onClick={openCreate}><Plus size={16} /> {t("admin.categories.add_top_level_category")}</button> : null}
        </div>
      </div>

      <section className="panel">
        {categories.length === 0 && !error ? (
          <EmptyState
            icon={FolderTree}
            title={t("admin.categories.no_categories")}
            hint={t("admin.categories.click_add_top_level_category_top_right_to")}
          />
        ) : (
          <div className="tree">
            {categories.map((cat) => (
              <TreeNode key={cat.id} category={cat} expanded={expanded} toggle={toggle} onAddSub={openAddSub} onEdit={openEdit} onDelete={remove} drag={drag} />
            ))}
          </div>
        )}
      </section>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={data.id ? t("admin.categories.edit_category") : data.parent_id ? t("admin.categories.add_subcategory") : t("admin.categories.add_top_level_category")}
        subtitle={t("admin.categories.top_level_categories_require_admin_privileges_subcategories_can")}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{t("admin.categories.cancel")}</button>
            <button className="button button-primary" onClick={submit} disabled={!data.name?.trim()}>{data.id ? t("admin.categories.save_changes") : t("admin.categories.create")}</button>
          </>
        }
      >
        <div className="field">
          <label>{t("admin.categories.name")}</label>
          <input value={data.name || ""} placeholder={t("admin.categories.e_g_tool_specification")} autoFocus onChange={(e) => setData({ ...data, name: e.target.value })} />
          {data.id ? <span className="field-hint">{t("admin.categories.id")} <span className="tree-keytag">{data.key}</span>{t("admin.categories.system_generated_immutable")}</span> : <span className="field-hint">{t("admin.categories.id_is_auto_generated_by_the_system")}</span>}
        </div>
        <div className="field">
          <label>{t("admin.categories.description")}</label>
          <input value={data.description || ""} placeholder={t("admin.categories.one_sentence_description_of_this_category")} onChange={(e) => setData({ ...data, description: e.target.value })} />
        </div>
        <div className="field">
          <label>{t("admin.categories.parent_category")}</label>
          <Combobox options={[...(isSuper ? [{ value: "", label: t("admin.categories.top_level_category") }] : []), ...parentOptions]} value={[data.parent_id || ""]} onChange={(v) => setData({ ...data, parent_id: v[0] || "" })} multiple={false} placeholder={t("admin.categories.select_parent_category")} />
        </div>
        <div className="field">
          <label>{t("admin.categories.icon")}</label>
          <IconPicker value={data.icon} onChange={(icon) => setData({ ...data, icon })} />
        </div>
        <div className="field">
          <label>{t("admin.categories.owning_team")}</label>
          <Combobox options={[{ value: "", label: t("admin.categories.none") }, ...teamOptions]} value={[data.responsible_team || ""]} onChange={(v) => setData({ ...data, responsible_team: v[0] || "" })} multiple={false} placeholder={t("admin.categories.select_owning_team")} />
        </div>
      </Modal>
    </AdminShell>
  );
}
