"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { getCategories, getTeams, createCategory, updateCategory, deleteCategory } from "@/lib/api";
import type { Category, Team } from "@/types/modex";

type FlatRow = { category: Category; depth: number };

function flatten(categories: Category[] | null | undefined, depth = 0): FlatRow[] {
  if (!categories) return [];
  return categories.flatMap((category) => [
    { category, depth },
    ...flatten(category.children || [], depth + 1),
  ]);
}

function AdminCategoryNode({
  category,
  teamOptions,
  onEdit,
  onAddSub,
  onDelete,
  depth = 0,
}: {
  category: Category;
  teamOptions: string[];
  onEdit: (cat: Category) => void;
  onAddSub: (parentId: string) => void;
  onDelete: (id: string, name: string) => void;
  depth?: number;
}) {
  return (
    <div>
      <div
        className="flex items-center gap-3 p-2 rounded border border-border bg-panel hover:bg-muted-panel"
        style={{ marginLeft: depth * 16 }}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium">{category.name}</span>
            <span className="code-chip text-xs">{category.key}</span>
            {category.icon && <span className="text-xs muted">[{category.icon}]</span>}
            {category.responsible_team && (
              <span className="tag text-xs bg-emerald-100 text-emerald-700">
                负责: {category.responsible_team}
              </span>
            )}
          </div>
          {category.description && (
            <div className="text-xs muted truncate mt-0.5">{category.description}</div>
          )}
        </div>
        <div className="flex items-center gap-1 text-xs">
          <button
            className="button"
            onClick={() => onAddSub(category.id)}
          >
            + 子领域
          </button>
          <button className="button" onClick={() => onEdit(category)}>
            编辑
          </button>
          <button
            className="button"
            onClick={() => onDelete(category.id, category.name)}
          >
            删除
          </button>
        </div>
      </div>
      {category.children && category.children.length > 0 && (
        <div>
          {category.children.map((child) => (
            <AdminCategoryNode
              key={child.id}
              category={child}
              teamOptions={teamOptions}
              onEdit={onEdit}
              onAddSub={onAddSub}
              onDelete={onDelete}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export default function AdminCategoriesPage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [error, setError] = useState("");
  const [rows, setRows] = useState<FlatRow[]>([]);

  // Modal for create/edit
  const [modalOpen, setModalOpen] = useState(false);
  const [modalData, setModalData] = useState<Partial<Category> & { id?: string }>({});

  // For parent selector in modal
  const [parentOptions, setParentOptions] = useState<FlatRow[]>([]);

  async function refresh() {
    try {
      const [tree, ts] = await Promise.all([getCategories(), getTeams()]);
      const safeTree = tree || [];
      setCategories(safeTree);
      setTeams(ts || []);
      setRows(flatten(safeTree));
      setParentOptions(flatten(safeTree)); // for parent select in modal
    } catch (e) {
      setError(String(e));
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  const teamOptions = (teams || []).map((t) => t.key);

  function openCreate() {
    setModalData({});
    setModalOpen(true);
    setError("");
  }

  function openAddSub(parentId: string) {
    setModalData({ parent_id: parentId });
    setModalOpen(true);
    setError("");
  }

  function openEdit(cat: Category) {
    setModalData({
      id: cat.id,
      key: cat.key,
      name: cat.name,
      description: cat.description,
      parent_id: cat.parent_id,
      icon: cat.icon,
      sort_order: cat.sort_order,
      responsible_team: cat.responsible_team,
    });
    setModalOpen(true);
    setError("");
  }

  function closeModal() {
    setModalOpen(false);
    setModalData({});
  }

  function updateModalField(field: string, value: any) {
    setModalData((prev) => ({ ...prev, [field]: value }));
  }

  async function submitModal() {
    setError("");
    try {
      const payload: Partial<Category> = {
        key: modalData.key,
        name: modalData.name || modalData.key,
        description: modalData.description || undefined,
        parent_id: modalData.parent_id || undefined,
        icon: modalData.icon || undefined,
        sort_order: modalData.sort_order ? Number(modalData.sort_order) : 0,
        responsible_team: modalData.responsible_team || undefined,
      };

      if (modalData.id) {
        // edit
        await updateCategory(modalData.id, payload);
      } else {
        await createCategory(payload);
      }
      closeModal();
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  async function remove(id: string, name: string) {
    if (!confirm(`删除分类「${name}」？子分类必须先删除。`)) return;
    setError("");
    try {
      await deleteCategory(id);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  const adminShellProps = {
    title: "分类 / 领域管理",
    kicker: "Domains & Hierarchy",
    description: "层级领域（Category）支持任意嵌套（实践 3-5 级，参考 Mintlify/GitBook 侧边栏）。可为领域指定「负责团队」（responsible_team），团队成员自动获得该领域下的管理权限。顶层领域创建需超管权限。",
  };

  return (
    <AdminShell {...adminShellProps}>
      {error ? (
        <div className="panel" style={{ borderColor: "#ef4444", color: "#b91c1c" }}>
          {error}
        </div>
      ) : null}

      {/* Header with create button */}
      <div className="flex items-center justify-between mb-3">
        <button className="button button-primary" onClick={openCreate}>
          新增顶级领域
        </button>
        <button className="button" onClick={refresh}>
          刷新
        </button>
      </div>

      {/* Tree view for domains */}
      <section className="panel">
        <div className="flex items-center justify-between mb-2">
          <h2 className="font-semibold">领域层级</h2>
        </div>

        {categories.length === 0 && !error && (
          <div className="empty-state mt-3">
            <div>
              <div className="font-semibold text-foreground">暂无领域</div>
              <p className="mt-1 text-sm">点击上方“新增顶级领域”开始创建层级结构。支持任意嵌套，可绑定负责团队。</p>
            </div>
          </div>
        )}

        {categories.length > 0 && (
          <div className="space-y-1">
            {categories.map((cat) => (
              <AdminCategoryNode
                key={cat.id}
                category={cat}
                teamOptions={teamOptions}
                onEdit={openEdit}
                onAddSub={openAddSub}
                onDelete={remove}
              />
            ))}
          </div>
        )}

        <div className="muted text-xs mt-3">
          说明：只有超级管理员可以创建顶层领域或管理团队绑定。设置 <code>SUPER_ADMIN_USERS</code> 环境变量获得首个超管后，即可从 0 开始创建领域层级，并将团队设置为负责人。
        </div>
      </section>

      {/* Modal for Create / Edit */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="panel w-full max-w-lg mx-4">
            <h2 className="font-semibold text-lg mb-4">
              {modalData.id ? "编辑领域" : "新增领域"}
            </h2>

            <div className="grid gap-3">
              <div>
                <label className="text-sm block mb-1">Key * (唯一)</label>
                <input
                  className="input w-full"
                  placeholder="如 standards"
                  value={modalData.key || ""}
                  onChange={(e) => updateModalField("key", e.target.value)}
                  disabled={!!modalData.id}
                />
              </div>
              <div>
                <label className="text-sm block mb-1">名称</label>
                <input
                  className="input w-full"
                  value={modalData.name || ""}
                  onChange={(e) => updateModalField("name", e.target.value)}
                />
              </div>
              <div>
                <label className="text-sm block mb-1">描述</label>
                <input
                  className="input w-full"
                  value={modalData.description || ""}
                  onChange={(e) => updateModalField("description", e.target.value)}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm block mb-1">父领域</label>
                  <select
                    className="input w-full"
                    value={modalData.parent_id || ""}
                    onChange={(e) => updateModalField("parent_id", e.target.value)}
                  >
                    <option value="">顶级领域</option>
                    {parentOptions.map((r) => (
                      <option key={r.category.id} value={r.category.id}>
                        {"— ".repeat(r.depth)}
                        {`${r.category.name} (${r.category.key})`}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="text-sm block mb-1">图标</label>
                  <select
                    className="input w-full"
                    value={modalData.icon || ""}
                    onChange={(e) => updateModalField("icon", e.target.value)}
                  >
                    <option value="">（默认图标）</option>
                    <option value="wrench">wrench - 工具/工程</option>
                    <option value="package">package - 模块/包</option>
                    <option value="box">box - 容器/内核</option>
                    <option value="layers">layers - 应用/多层</option>
                    <option value="layout">layout - 前端/布局</option>
                    <option value="book-open">book-open - 文档/框架</option>
                    <option value="book">book - 规范/教程</option>
                    <option value="cpu">cpu - 核心/计算</option>
                    <option value="git-branch">git-branch - 版本控制</option>
                    <option value="tool">tool - 工具链</option>
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm block mb-1">排序</label>
                  <input
                    className="input w-full"
                    type="number"
                    value={modalData.sort_order ?? 0}
                    onChange={(e) => updateModalField("sort_order", e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-sm block mb-1">负责团队</label>
                  <select
                    className="input w-full"
                    value={modalData.responsible_team || ""}
                    onChange={(e) => updateModalField("responsible_team", e.target.value)}
                  >
                    <option value="">无</option>
                    {teamOptions.map((k) => (
                      <option key={k} value={k}>{k}</option>
                    ))}
                  </select>
                </div>
              </div>
            </div>

            <div className="flex gap-2 mt-6">
              <button className="button button-primary flex-1" onClick={submitModal}>
                {modalData.id ? "保存修改" : "创建"}
              </button>
              <button className="button flex-1" onClick={closeModal}>
                取消
              </button>
            </div>

            <p className="text-xs muted mt-3">
              顶层领域需超管权限，子领域可由父领域管理员或负责团队创建。
            </p>
          </div>
        </div>
      )}
    </AdminShell>
  );
}
