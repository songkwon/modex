export type Category = {
  id: string;
  key: string;
  name: string;
  description: string;
  icon?: string;
  parent_id?: string;
  sort_order?: number;
  responsible_team?: string;
  children?: Category[];
};

export type ModuleInfo = {
  module_key: string;
  name: string;
  description: string;
  owner_group: string;
  repo_type?: string;
  repo_url: string;
  default_version: string;
  status: string;
  package_version: string;
  channel: string;
  edition: string;
  keywords: string[];
  maintainers: string[];
  category_ids?: string[];
  category_path: string;

  // GitLab integration fields (for CI-driven sync like Mintlify)
  source_type?: string;      // "gitlab" | "manual"
  doc_type?: string;         // vitepress | vuepress | fumadocs | markdown
  mount?: string;            // "single" | "split"
  gitlab_branch?: string;
  gitlab_path?: string;      // subdir in repo, e.g. "docs/standard"
  last_synced_commit?: string;
  last_synced_at?: string;
  deploy_token_set?: boolean;

  updated_at: string;
  reads_7d: number;
  reads_30d: number;
};

export type User = {
  id: string;
  username: string;
  display_name: string;
  email: string;
  department: string;
  avatar?: string;
  roles?: string[];
  managed_categories?: string[];
  is_super_admin?: boolean;
  is_team_admin?: boolean;
  source?: string;
  status?: string;
  last_login_at?: string;
  created_at?: string;
  updated_at?: string;
};

// Team = 文档维护团队. Leader (负责人) can add/remove members ("拉人").
// Can be assigned as responsible_team on a Category (领域) to own its doc structure & modules.
export type Team = {
  id: string;
  key: string;
  name: string;
  description?: string;
  leaders: string[];
  members?: string[];
  created_at?: string;
  updated_at?: string;
};

export type AuthConfig = {
  oidc_login_enabled: boolean;
  login_url: string;
  frontend_base_url: string;
  auto_login?: boolean;
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
  breadcrumb: string;
  entry_key: string;
  match_terms: string[];
};

export type AskResponse = {
  query: string;
  answer: string;
  provider: string;
  sources: SearchResult[];
};

export type SearchResponse = {
  query: string;
  mode: string;
  total: number;
  results: SearchResult[];
  facets: Record<string, Record<string, number>>;
};
