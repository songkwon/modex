export type Category = {
  id: string;
  key: string;
  name: string;
  description: string;
  children?: Category[];
};

export type ModuleInfo = {
  module_key: string;
  name: string;
  description: string;
  owner_group: string;
  repo_url: string;
  default_version: string;
  status: string;
  package_version: string;
  channel: string;
  edition: string;
  keywords: string[];
  maintainers: string[];
  category_path: string;
  updated_at: string;
  reads_7d: number;
  reads_30d: number;
};

export type User = {
  username: string;
  display_name: string;
  email: string;
  department: string;
};

export type AuthConfig = {
  auth_mode: "mock" | "oidc" | string;
  oidc_login_enabled: boolean;
  login_url: string;
  frontend_base_url: string;
};

export type SearchResult = {
  doc_id: string;
  title: string;
  snippet: string;
  path: string;
  score: number;
  search_mode: string;
  module_key: string;
  module_name: string;
  docs_version: string;
  package_version: string;
  entry_type: string;
  owner_group: string;
  status: string;
  updated_at: string;
  keywords: string[];
};

export type SearchResponse = {
  query: string;
  mode: string;
  total: number;
  results: SearchResult[];
  facets: Record<string, Record<string, number>>;
};

export type PageStat = {
  doc_id: string;
  title: string;
  module_key: string;
  module_name: string;
  docs_version: string;
  path: string;
  pv: number;
  uv: number;
  reads_7d: number;
  reads_30d: number;
  avg_duration_seconds: number;
  last_viewed_at: string;
};

export type AnalyticsPages = {
  popular_pages: PageStat[];
  total_pv: number;
  reads_7d: number;
  events: string[];
};
