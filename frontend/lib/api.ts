import type { AnalyticsPages, AuthConfig, Category, ModuleInfo, SearchResponse, User } from "@/types/modex";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8671";

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
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
export const mockLogin = () => api<{ user: User }>("/api/auth/mock-login", { method: "POST", body: "{}" });
export const logout = () => api<{ ok: boolean }>("/api/auth/logout", { method: "POST", body: "{}" });
export const getAuthConfig = () => api<AuthConfig>("/api/config");
export const getCategories = () => api<Category[]>("/api/categories/tree");
export const getModules = (query = "") => api<ModuleInfo[]>(`/api/modules${query}`);
export const getModule = (moduleKey: string) => api<ModuleInfo>(`/api/modules/${moduleKey}/info`);
export const getEntries = (moduleKey: string, version: string) => api<any[]>(`/api/modules/${moduleKey}/versions/${version}/entries`);
export const getPage = (moduleKey: string, version: string, entry: string) => api<any>(`/api/docs/${moduleKey}/${version}/${entry}`);
export const searchDocs = (body: unknown) => api<SearchResponse>("/api/search", { method: "POST", body: JSON.stringify(body) });

export const recordPageView = (body: { doc_id: string; session_id: string; duration_seconds?: number; scroll_depth?: number }) =>
  api<{ status: string; view_id: string }>("/api/analytics/page-view", { method: "POST", body: JSON.stringify(body) });

export const recordReadProgress = (body: { doc_id: string; session_id: string; duration_seconds: number; scroll_depth: number }) =>
  api<{ status: string }>("/api/analytics/read-progress", { method: "POST", body: JSON.stringify(body) });

export const getPageAnalytics = () => api<AnalyticsPages>("/api/admin/analytics/pages");
