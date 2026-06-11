"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { ChevronRight, GripVertical, Pencil, Plus, Trash2 } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import { CategoryIcon, IconPicker } from "@/components/ui/icon-picker";
import { getCategories, getTeams, createCategory, updateCategory, deleteCategory, moveCategory } from "@/lib/api";
import type { Category, Team } from "@/types/modex";

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
        <span className="tree-grip" title="拖动排序 / 调整层级"><GripVertical size={14} /></span>
        {hasChildren ? (
          <span className={`tree-twist${open ? " open" : ""}`} onClick={() => toggle(category.id)}>
            <ChevronRight size={15} />
          </span>
        ) : (
          <span className="tree-twist" style={{ visibility: "hidden" }}><ChevronRight size={15} /></span>
        )}
        <span className="tree-icon"><CategoryIcon name={category.icon} /></span>
        <span className="tree-label">{category.name}</span>
        <span className="tree-keytag">{category.key}</span>
        {category.responsible_team ? <span className="badge badge-success">负责: {category.responsible_team}</span> : null}
        {category.description ? <span className="muted" style={{ fontSize: 12, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", maxWidth: 240 }}>{category.description}</span> : null}
        <span className="tree-spacer" />
        <div className="row-actions">
          <button className="icon-btn" onClick={() => onAddSub(category.id)} title="新增子领域"><Plus size={14} /></button>
          <button className="icon-btn" onClick={() => onEdit(category)} title="编辑"><Pencil size={14} /></button>
          <button className="icon-btn danger" onClick={() => onDelete(category.id, category.name)} title="删除"><Trash2 size={14} /></button>
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
  const [categories, setCategories] = useState<Category[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [error, setError] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [modalOpen, setModalOpen] = useState(false);
  const [data, setData] = useState<Partial<Category> & { id?: string }>({});

  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<DropTarget | null>(null);
  const byId = useRef<Map<string, Category>>(new Map());

  async function refresh() {
    try {
      const [tree, ts] = await Promise.all([getCategories(), getTeams()]);
      const safe = tree || [];
      setCategories(safe);
      setTeams(ts || []);
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
    if (!confirm(`删除分类「${name}」？子分类必须先删除。`)) return;
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
      title="分类 / 领域管理"
      kicker="Domains & Hierarchy"
      description="层级领域支持任意嵌套。直接在树上拖动即可调整顺序与层级；可为领域指定负责团队，成员自动获得该领域下的管理权限。"
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <div className="admin-toolbar">
        <div className="muted" style={{ fontSize: 13 }}>{flatten(categories).length} 个领域节点 · 拖动卡片排序或改层级</div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}><Plus size={16} /> 新增顶级领域</button>
        </div>
      </div>

      <section className="panel">
        {categories.length === 0 && !error ? (
          <div className="empty-state">
            <div>
              <div style={{ fontWeight: 640, color: "hsl(var(--foreground))" }}>暂无领域</div>
              <p style={{ marginTop: 4, fontSize: 13 }}>点击「新增顶级领域」开始创建层级结构。支持任意嵌套，可绑定负责团队。</p>
            </div>
          </div>
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
        title={data.id ? "编辑领域" : data.parent_id ? "新增子领域" : "新增顶级领域"}
        subtitle="顶层领域需超管权限，子领域可由父领域管理员或负责团队创建"
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>取消</button>
            <button className="button button-primary" onClick={submit} disabled={!data.name?.trim()}>{data.id ? "保存修改" : "创建"}</button>
          </>
        }
      >
        <div className="field">
          <label>名称 *</label>
          <input value={data.name || ""} placeholder="如 工具规范" autoFocus onChange={(e) => setData({ ...data, name: e.target.value })} />
          {data.id ? <span className="field-hint">标识 <span className="tree-keytag">{data.key}</span>（系统生成，不可修改）</span> : <span className="field-hint">标识由系统自动生成。</span>}
        </div>
        <div className="field">
          <label>描述</label>
          <input value={data.description || ""} placeholder="一句话说明该领域" onChange={(e) => setData({ ...data, description: e.target.value })} />
        </div>
        <div className="field">
          <label>父领域</label>
          <Combobox options={[{ value: "", label: "（顶级领域）" }, ...parentOptions]} value={[data.parent_id || ""]} onChange={(v) => setData({ ...data, parent_id: v[0] || "" })} multiple={false} placeholder="选择父领域…" />
        </div>
        <div className="field">
          <label>图标</label>
          <IconPicker value={data.icon} onChange={(icon) => setData({ ...data, icon })} />
        </div>
        <div className="field">
          <label>负责团队</label>
          <Combobox options={[{ value: "", label: "（无）" }, ...teamOptions]} value={[data.responsible_team || ""]} onChange={(v) => setData({ ...data, responsible_team: v[0] || "" })} multiple={false} placeholder="选择负责团队…" />
        </div>
      </Modal>
    </AdminShell>
  );
}
