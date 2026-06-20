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
        <span className="tree-grip" title={t("legacy.13084b6a4ce9")}><GripVertical size={14} /></span>
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
          <button className="icon-btn" onClick={() => onAddSub(category.id)} title={t("legacy.447e2db4681e")}><Plus size={14} /></button>
          <button className="icon-btn" onClick={() => onEdit(category)} title={t("legacy.051836569928")}><Pencil size={14} /></button>
          <button className="icon-btn danger" onClick={() => onDelete(category.id, category.name)} title={t("legacy.2f9daa828907")}><Trash2 size={14} /></button>
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
    if (!confirm(t("legacy.b4976b17b23a", { value1: name }))) return;
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
      title={t("legacy.62f8edc30321")}
      kicker="Categories"
      description={t("legacy.f0315110a4b8")}
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <div className="admin-toolbar">
        <div className="muted" style={{ fontSize: 13 }}>{flatten(categories).length} {t("legacy.4f2370f5e386")}</div>
        <div className="admin-toolbar-actions">
          {isSuper ? <button className="button button-primary" onClick={openCreate}><Plus size={16} /> {t("legacy.ffabe4e5b684")}</button> : null}
        </div>
      </div>

      <section className="panel">
        {categories.length === 0 && !error ? (
          <EmptyState
            icon={FolderTree}
            title={t("legacy.3eeb3b76eef2")}
            hint={t("legacy.0fe8c5284dd5")}
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
        title={data.id ? t("legacy.182a0480aae8") : data.parent_id ? t("legacy.447e2db4681e") : t("legacy.ffabe4e5b684")}
        subtitle={t("legacy.c1eab53f8cfb")}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{t("legacy.2cd0f3be8738")}</button>
            <button className="button button-primary" onClick={submit} disabled={!data.name?.trim()}>{data.id ? t("legacy.991bb7cfe5a8") : t("legacy.cde2cd071d25")}</button>
          </>
        }
      >
        <div className="field">
          <label>{t("legacy.909712f2847d")}</label>
          <input value={data.name || ""} placeholder={t("legacy.494b23115e71")} autoFocus onChange={(e) => setData({ ...data, name: e.target.value })} />
          {data.id ? <span className="field-hint">{t("legacy.301c9461c1d0")} <span className="tree-keytag">{data.key}</span>{t("legacy.e7d3ed9636ac")}</span> : <span className="field-hint">{t("legacy.de9b50b07a8c")}</span>}
        </div>
        <div className="field">
          <label>{t("legacy.dc2ba467fc7a")}</label>
          <input value={data.description || ""} placeholder={t("legacy.52b4136bf071")} onChange={(e) => setData({ ...data, description: e.target.value })} />
        </div>
        <div className="field">
          <label>{t("legacy.e77b87e5b8e7")}</label>
          <Combobox options={[...(isSuper ? [{ value: "", label: t("legacy.76362d22d383") }] : []), ...parentOptions]} value={[data.parent_id || ""]} onChange={(v) => setData({ ...data, parent_id: v[0] || "" })} multiple={false} placeholder={t("legacy.6f6d88776984")} />
        </div>
        <div className="field">
          <label>{t("legacy.0d720eeea264")}</label>
          <IconPicker value={data.icon} onChange={(icon) => setData({ ...data, icon })} />
        </div>
        <div className="field">
          <label>{t("legacy.21fa2b2c4376")}</label>
          <Combobox options={[{ value: "", label: t("legacy.c7bcc6d27f3a") }, ...teamOptions]} value={[data.responsible_team || ""]} onChange={(v) => setData({ ...data, responsible_team: v[0] || "" })} multiple={false} placeholder={t("legacy.66c48e3b9eb2")} />
        </div>
      </Modal>
    </AdminShell>
  );
}
