import type { AskResponse, AuthConfig, Category, Group, ModuleInfo, SearchResponse, Team, User } from "@/types/modex";

const PUBLIC_API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8671";
const SERVER_API_BASE = process.env.INTERNAL_API_BASE_URL || PUBLIC_API_BASE;

function apiBaseURL() {
  return typeof window === "undefined" ? SERVER_API_BASE : PUBLIC_API_BASE;
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${apiBaseURL()}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {})
    },
    cache: "no-store"
  });
  if (!res.ok) {
    throw new Error(`API ${res.status}: ${await res.text()}`);
  }
  return res.json();
}

export const getMe = () => api<User>("/api/auth/me");
export const getOptionalMe = () => api<User | null>("/api/auth/me?optional=1");
export const mockLogin = (username?: string) =>
  api<{ user: User }>("/api/auth/mock-login", { method: "POST", body: JSON.stringify(username ? { username } : {}) });
export const logout = () => api<{ ok: boolean }>("/api/auth/logout", { method: "POST", body: "{}" });
export const getAuthConfig = () => api<AuthConfig>("/api/config");
export const getCategories = () => api<Category[]>("/api/categories/tree");
// Admin views: only the category subtrees the current user may manage (full tree for super admins).
export const getManagedCategories = () => api<Category[]>("/api/categories/tree?scope=managed");
export const getModules = (query = "") => api<ModuleInfo[]>(`/api/modules${query}`);
export const updateModule = (moduleKey: string, body: Partial<ModuleInfo> & { deploy_token?: string }) => api<ModuleInfo>(`/api/admin/modules/${moduleKey}`, { method: "PUT", body: JSON.stringify(body) });
export const createModule = (body: Partial<ModuleInfo>) => api<ModuleInfo>("/api/admin/modules", { method: "POST", body: JSON.stringify(body) });
export const getDeployToken = (moduleKey: string) => api<{ deploy_token: string; deploy_url: string; module_key: string }>(`/api/admin/modules/${moduleKey}/deploy-token`);
export const rotateDeployToken = (moduleKey: string) => api<{ deploy_token: string; deploy_url: string; module_key: string }>(`/api/admin/modules/${moduleKey}/deploy-token`, { method: "POST", body: "{}" });
export const getModule = (moduleKey: string) => api<ModuleInfo>(`/api/modules/${moduleKey}/info`);
export const getEntries = (moduleKey: string, version: string) => api<any[]>(`/api/modules/${moduleKey}/versions/${version}/entries`);
export const getModuleVersions = (moduleKey: string) => api<{ docs_version: string; display_name?: string; is_default?: boolean }[]>(`/api/modules/${moduleKey}/versions`);
export const getNav = (moduleKey: string, version: string, entry: string) => api<{ title: string; path: string; children?: { title: string; path: string }[] }[]>(`/api/docs/${moduleKey}/${version}/${entry}/nav`);
export const getPage = (moduleKey: string, version: string, entry: string) => api<any>(`/api/docs/${moduleKey}/${version}/${entry}`);
export const searchDocs = (body: unknown) => api<SearchResponse>("/api/search", { method: "POST", body: JSON.stringify(body) });
export const askAI = (query: string, scope?: { module_key?: string; category_ids?: string[] }) =>
  api<AskResponse>("/api/ask", { method: "POST", body: JSON.stringify({ query, ...(scope || {}) }) });

export const recordDocFeedback = (body: { doc_id: string; rating: "good" | "bad"; session_id: string; comment?: string }) =>
  api<{ status: string }>("/api/analytics/feedback", { method: "POST", body: JSON.stringify(body) });
export const recordPageView = (body: { doc_id: string; session_id: string; read_id: string }) =>
  api<{ status: string }>("/api/analytics/page-view", { method: "POST", body: JSON.stringify(body) });
export const recordReadProgress = (body: { doc_id: string; session_id: string; read_id: string; duration_seconds: number; scroll_depth: number }) =>
  api<{ status: string }>("/api/analytics/read-progress", { method: "POST", body: JSON.stringify(body) });

export type DocReadStats = {
  doc_id: string;
  total: number;
  avg_duration_seconds: number;
  daily: { date: string; count: number }[];
  readers: { reader: string; user_id: string; count: number; avg_duration_seconds: number; last_read_at: string }[];
};
export const getDocAnalytics = (docId: string, days = 30) =>
  api<{ source: "posthog" | "builtin"; stats: DocReadStats }>(
    `/api/analytics/doc?doc_id=${encodeURIComponent(docId)}&days=${days}`
  );

export type UserFavorite = { id: string; user_id: string; module_key: string; created_at: string };
export type RecentDoc = {
  id?: string;
  user_id?: string;
  doc_id: string;
  title: string;
  module_key: string;
  module_name: string;
  docs_version: string;
  entry_key: string;
  href: string;
  viewed_at?: string;
};
export const getMyFavorites = () => api<{ favorites: UserFavorite[] }>("/api/me/favorites");
export const setMyFavorite = (moduleKey: string, favorite: boolean) =>
  api<{ favorites: UserFavorite[] }>("/api/me/favorites", { method: "POST", body: JSON.stringify({ module_key: moduleKey, favorite }) });
export const getMyRecentDocs = () => api<{ recent: RecentDoc[] }>("/api/me/recent");
export const recordMyRecentDoc = (recent: RecentDoc) =>
  api<{ recent: RecentDoc }>("/api/me/recent", { method: "POST", body: JSON.stringify(recent) });

export const getUsers = (keyword = "") => api<User[]>(`/api/admin/users${keyword ? `?keyword=${encodeURIComponent(keyword)}` : ""}`);
export const createUser = (body: Partial<User>) => api<User>("/api/admin/users", { method: "POST", body: JSON.stringify(body) });
export const updateUser = (id: string, body: Partial<User>) => api<User>(`/api/admin/users/${id}`, { method: "PUT", body: JSON.stringify(body) });
export const deleteUser = (id: string) => api<{ status: string }>(`/api/admin/users/${id}`, { method: "DELETE" });
export const getGroups = () => api<Group[]>("/api/admin/groups");

export const getTeams = () => api<Team[]>("/api/admin/teams");
export const createTeam = (body: Partial<Team>) => api<Team>("/api/admin/teams", { method: "POST", body: JSON.stringify(body) });
export const updateTeam = (key: string, body: Partial<Team>) => api<Team>(`/api/admin/teams/${key}`, { method: "PUT", body: JSON.stringify(body) });
export const deleteTeam = (key: string) => api<{ status: string }>(`/api/admin/teams/${key}`, { method: "DELETE" });
export const addTeamMember = (key: string, username: string) =>
  api<Team>(`/api/admin/teams/${key}/members`, { method: "POST", body: JSON.stringify({ username }) });
export const removeTeamMember = (key: string, username: string) =>
  api<Team>(`/api/admin/teams/${key}/members/${encodeURIComponent(username)}`, { method: "DELETE" });

// Admin category mutations (for 领域/分类 ownership + hierarchy mgmt). Public tree is /api/categories/tree.
export const createCategory = (body: Partial<Category>) => api<Category>("/api/admin/categories", { method: "POST", body: JSON.stringify(body) });
export const updateCategory = (id: string, body: Partial<Category>) => api<Category>(`/api/admin/categories/${id}`, { method: "PUT", body: JSON.stringify(body) });
export const deleteCategory = (id: string) => api<{ status: string; id: string }>(`/api/admin/categories/${id}`, { method: "DELETE" });
// Drag-and-drop move: reparent + position among siblings. index is 0-based.
export const moveCategory = (id: string, body: { parent_id: string; index: number }) =>
  api<Category>(`/api/admin/categories/${id}/move`, { method: "POST", body: JSON.stringify(body) });

export const reindexSearch = () => api<Record<string, unknown>>("/api/search/reindex", { method: "POST", body: "{}" });
export const reindexEmbeddings = () => api<Record<string, unknown>>("/api/embeddings/reindex", { method: "POST", body: "{}" });

// Admin platform settings (AI model connection). Key is write-only; reads return ask_api_key_set.
export type AISettings = {
  ask_protocol?: string;
  ask_base_url?: string;
  ask_model?: string;
  ask_api_key?: string;
  ask_system_prompt?: string;
  ask_max_tokens?: number;
  ask_temperature?: number;
  embedding_base_url?: string;
  embedding_model?: string;
  embedding_api_key?: string;
  embedding_dim?: number;
  rerank_base_url?: string;
  rerank_model?: string;
  rerank_api_key?: string;
  rerank_top_k?: number;
  chunk_strategy?: string;
  chunk_size?: number;
  chunk_overlap?: number;
  recall_test_query?: string;
  recall_test_top_k?: number;
  recall_test_doc_ids?: string;
  updated_at?: string;
};
export type PlatformSettings = {
  ai: AISettings;
  ask_api_key_set: boolean;
  embedding_api_key_set?: boolean;
  rerank_api_key_set?: boolean;
  ask_system_prompt_default: string;
};
export const getSettings = () => api<PlatformSettings>("/api/admin/settings");
export const saveSettings = (ai: AISettings) => api<PlatformSettings>("/api/admin/settings", { method: "PUT", body: JSON.stringify(ai) });
export const fetchModels = (base_url: string, api_key?: string, protocol?: string) =>
  api<{ models: string[] }>("/api/admin/settings/models", { method: "POST", body: JSON.stringify({ base_url, api_key: api_key || "", protocol: protocol || "" }) });

export type ConnectedApp = {
  id: string;
  name: string;
  description?: string;
  client_id: string;
  client_secret?: string;
  redirect_uris: string[];
  scopes: string[];
  trusted: boolean;
  enabled: boolean;
  created_by?: string;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
};
export type ConnectedAppDraft = Pick<ConnectedApp, "name" | "description" | "redirect_uris" | "scopes" | "trusted" | "enabled"> & {
  client_id?: string;
};
export const getConnectedApps = () => api<{ apps: ConnectedApp[] }>("/api/admin/connected-apps");
export const createConnectedApp = (body: ConnectedAppDraft) =>
  api<ConnectedApp>("/api/admin/connected-apps", { method: "POST", body: JSON.stringify(body) });
export const updateConnectedApp = (id: string, body: ConnectedAppDraft) =>
  api<ConnectedApp>(`/api/admin/connected-apps/${id}`, { method: "PUT", body: JSON.stringify(body) });
export const deleteConnectedApp = (id: string) => api<{ status: string }>(`/api/admin/connected-apps/${id}`, { method: "DELETE" });

export type RecallTestResult = {
  query: string;
  top_k: number;
  expected_doc_ids: string[];
  actual_doc_ids: string[];
  recall_at_k: number;
  mrr: number;
  ndcg: number;
  note?: string;
  results?: { doc_id: string; title: string; score: number; path: string }[];
};
export const runRecallTest = () => api<RecallTestResult>("/api/admin/settings/recall-test", { method: "POST", body: "{}" });

// Doc-engine plugin registry (built-in, admin-managed). Effective enabled+config
// is also served (un-gated) on /api/config so the renderer can read it.
export type PluginSetting = { enabled: boolean; config?: Record<string, string> };
export type PluginConfig = Record<string, PluginSetting>;
export type PluginField = { key: string; label: string; placeholder?: string; default?: string };
export type PluginState = {
  key: string;
  name: string;
  description: string;
  category: string;
  default_enabled: boolean;
  fields?: PluginField[];
  enabled: boolean;
  config: Record<string, string>;
  // Set for admin-imported plugins.
  uploaded?: boolean;
  kind?: "component" | "fence";
  tag?: string;
  lang?: string;
  code?: string;
};
export const getPlugins = () => api<{ plugins: PluginState[] }>("/api/admin/plugins");
export const savePlugins = (plugins: PluginConfig) =>
  api<{ plugins: PluginState[] }>("/api/admin/plugins", { method: "PUT", body: JSON.stringify({ plugins }) });
// Effective plugin config for the doc renderer (subset of /api/config).
export const getDocsPluginConfig = () => api<{ plugins?: PluginConfig }>("/api/config");

// Admin-imported, sandbox-rendered JSX plugins.
export type UploadedPlugin = {
  key: string;
  name: string;
  description?: string;
  category?: string;
  kind: "component" | "fence";
  tag?: string;
  lang?: string;
  code: string;
  format?: string;
  version?: number;
};
export const importPlugin = (p: UploadedPlugin) =>
  api<{ plugin: UploadedPlugin; plugins: PluginState[] }>("/api/admin/plugins/import", { method: "POST", body: JSON.stringify(p) });
export const deletePlugin = (key: string) =>
  api<{ status: string; plugins: PluginState[] }>(`/api/admin/plugins/import/${encodeURIComponent(key)}`, { method: "DELETE" });
// Enabled uploaded plugins for the renderer (includes JSX source).
export const getDocsUploadedPlugins = () => api<{ plugins: UploadedPlugin[] }>("/api/docs/plugins");

// Reusable snippets + variables (Mintlify-style partials).
export type Snippet = { key: string; name: string; content: string };
export type SnippetData = { snippets: Snippet[]; variables: Record<string, string> };
export const getSnippets = () => api<SnippetData>("/api/admin/snippets");
export const saveSnippets = (data: SnippetData) =>
  api<SnippetData>("/api/admin/snippets", { method: "PUT", body: JSON.stringify(data) });
// Read-only copy for the renderer (un-gated, no secrets).
export const getDocsSnippets = () => api<SnippetData>("/api/docs/snippets");

export const getMCPToken = () => api<{ mcp_token: string }>("/api/me/mcp-token");
export const rotateMCPToken = () => api<{ mcp_token: string }>("/api/me/mcp-token", { method: "POST", body: "{}" });
