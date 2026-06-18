CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  email TEXT,
  department TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS groups (
  id TEXT PRIMARY KEY,
  group_key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  source TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

-- Teams: document maintenance teams with leader + members.
-- A team can be assigned as responsible_team on docs_category (领域 owner).
-- Members/leaders get implicit management rights on assigned domains (see canManageViaResponsibleTeam).
CREATE TABLE IF NOT EXISTS teams (
  id TEXT PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT,
  leader TEXT,
  members JSONB DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_groups (
  user_id TEXT REFERENCES users(id),
  group_id TEXT REFERENCES groups(id),
  PRIMARY KEY (user_id, group_id)
);

CREATE TABLE IF NOT EXISTS connected_app (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  client_id TEXT NOT NULL UNIQUE,
  client_secret_hash TEXT,
  redirect_uris JSONB DEFAULT '[]'::jsonb,
  scopes JSONB DEFAULT '[]'::jsonb,
  trusted BOOLEAN DEFAULT false,
  enabled BOOLEAN DEFAULT true,
  created_by TEXT REFERENCES users(id),
  last_used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_grant (
  id TEXT PRIMARY KEY,
  app_id TEXT REFERENCES connected_app(id),
  user_id TEXT REFERENCES users(id),
  code_hash TEXT,
  access_token_hash TEXT,
  refresh_token_hash TEXT,
  redirect_uri TEXT,
  scopes JSONB DEFAULT '[]'::jsonb,
  code_expires_at TIMESTAMPTZ,
  access_expires_at TIMESTAMPTZ,
  refresh_expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth_grant_code_hash ON oauth_grant(code_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_grant_access_token_hash ON oauth_grant(access_token_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_grant_refresh_token_hash ON oauth_grant(refresh_token_hash);

CREATE TABLE IF NOT EXISTS docs_category (
  id TEXT PRIMARY KEY,
  parent_id TEXT REFERENCES docs_category(id),
  key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT,
  icon TEXT,
  sort_order INT DEFAULT 0,
  status TEXT DEFAULT 'active',
  responsible_team TEXT,  -- team key owning this 领域 (domain); members get mgmt rights
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_module (
  id TEXT PRIMARY KEY,
  module_key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT,
  owner_group TEXT,
  repo_type TEXT,
  repo_url TEXT,
  default_version_id TEXT,
  visibility TEXT DEFAULT 'internal',
  status TEXT DEFAULT 'active',
  package_name TEXT,
  package_version TEXT,
  channel TEXT,
  edition TEXT,
  keywords JSONB DEFAULT '[]'::jsonb,
  maintainers JSONB DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_module_category (
  module_id TEXT REFERENCES docs_module(id),
  category_id TEXT REFERENCES docs_category(id),
  is_primary BOOLEAN DEFAULT false,
  PRIMARY KEY (module_id, category_id)
);

CREATE TABLE IF NOT EXISTS docs_version (
  id TEXT PRIMARY KEY,
  module_id TEXT REFERENCES docs_module(id),
  docs_version TEXT NOT NULL,
  display_name TEXT,
  version_type TEXT,
  is_default BOOLEAN DEFAULT false,
  status TEXT DEFAULT 'active',
  source_branch TEXT,
  package_version TEXT,
  channel TEXT,
  edition TEXT,
  support_status TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(module_id, docs_version)
);

CREATE TABLE IF NOT EXISTS docs_entry (
  id TEXT PRIMARY KEY,
  module_id TEXT REFERENCES docs_module(id),
  version_id TEXT REFERENCES docs_version(id),
  entry_key TEXT NOT NULL,
  title TEXT NOT NULL,
  entry_type TEXT NOT NULL,
  builder TEXT,
  source TEXT,
  storage_uri TEXT,
  nav_uri TEXT,
  index_status TEXT,
  is_primary BOOLEAN DEFAULT false,
  sort_order INT DEFAULT 0,
  status TEXT DEFAULT 'active',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_release (
  id TEXT PRIMARY KEY,
  module_id TEXT REFERENCES docs_module(id),
  version_id TEXT REFERENCES docs_version(id),
  release_id TEXT NOT NULL UNIQUE,
  commit_sha TEXT,
  branch TEXT,
  publisher TEXT,
  pipeline_url TEXT,
  build_system TEXT,
  build_id TEXT,
  artifact_version TEXT,
  package_version TEXT,
  storage_uri TEXT,
  status TEXT,
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_page (
  id TEXT PRIMARY KEY,
  module_id TEXT REFERENCES docs_module(id),
  version_id TEXT REFERENCES docs_version(id),
  entry_id TEXT REFERENCES docs_entry(id),
  release_id TEXT REFERENCES docs_release(id),
  doc_id TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  description TEXT,
  path TEXT,
  source_file TEXT,
  doc_type TEXT,
  status TEXT DEFAULT 'active',
  owner_group TEXT,
  tags JSONB DEFAULT '[]'::jsonb,
  content_text TEXT,
  updated_at TIMESTAMPTZ,
  last_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_page_view (
  id BIGSERIAL PRIMARY KEY,
  page_id TEXT REFERENCES docs_page(id),
  module_id TEXT REFERENCES docs_module(id),
  version_id TEXT REFERENCES docs_version(id),
  user_id TEXT REFERENCES users(id),
  session_id TEXT,
  duration_seconds INT,
  scroll_depth NUMERIC,
  viewed_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_search_log (
  id BIGSERIAL PRIMARY KEY,
  user_id TEXT,
  query TEXT,
  mode TEXT,
  filters_json JSONB,
  result_count INT,
  clicked_doc_id TEXT,
  searched_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_embedding (
  id BIGSERIAL PRIMARY KEY,
  page_id TEXT REFERENCES docs_page(id),
  doc_id TEXT,
  chunk_id TEXT,
  module_id TEXT REFERENCES docs_module(id),
  version_id TEXT REFERENCES docs_version(id),
  entry_id TEXT REFERENCES docs_entry(id),
  content TEXT,
  embedding vector(384),
  embedding_json JSONB,
  metadata_json JSONB,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_mcp_log (
  id BIGSERIAL PRIMARY KEY,
  tool_name TEXT,
  user_id TEXT,
  query TEXT,
  input_json JSONB,
  result_count INT,
  created_at TIMESTAMPTZ DEFAULT now()
);
