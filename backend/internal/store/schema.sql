CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS auth_session (
  key TEXT PRIMARY KEY,
  value_json JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  email TEXT,
  department TEXT,
  avatar TEXT,
  roles_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  managed_categories_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  source TEXT,
  status TEXT,
  is_super_admin BOOLEAN NOT NULL DEFAULT false,
  mcp_token TEXT,
  last_login_at TIMESTAMPTZ,
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
  leaders JSONB NOT NULL DEFAULT '[]'::jsonb,
  members JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
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

-- Built-in public OAuth client for Codex MCP OAuth login. Codex CLI supports
-- `--oauth-client-id` but does not take a client secret for MCP login, so this
-- client intentionally has an empty secret and only allows local loopback
-- callbacks.
INSERT INTO connected_app(id,name,description,client_id,client_secret_hash,redirect_uris,scopes,trusted,enabled,created_at,updated_at)
VALUES(
  'app-codex-cli',
  'Codex CLI MCP OAuth',
  'Built-in public OAuth client for Codex MCP login.',
  'codex-cli',
  '',
  '["http://localhost","http://127.0.0.1","http://[::1]"]'::jsonb,
  '["modex:mcp:read","modex:docs:read"]'::jsonb,
  true,
  true,
  now(),
  now()
)
ON CONFLICT(client_id) DO NOTHING;

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
  default_version TEXT,
  visibility TEXT DEFAULT 'internal',
  status TEXT DEFAULT 'active',
  package_name TEXT,
  package_version TEXT,
  channel TEXT,
  edition TEXT,
  keywords JSONB DEFAULT '[]'::jsonb,
  maintainers JSONB DEFAULT '[]'::jsonb,
  category_path TEXT,
  source_type TEXT,
  doc_type TEXT,
  mount TEXT,
  gitlab_branch TEXT,
  gitlab_path TEXT,
  deploy_token TEXT,
  last_synced_commit TEXT,
  last_synced_at TIMESTAMPTZ,
  reads_7d INT NOT NULL DEFAULT 0,
  reads_30d INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS docs_module_deploy_token_uidx
  ON docs_module(deploy_token)
  WHERE deploy_token IS NOT NULL AND deploy_token <> '';

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
  trigger_type TEXT,
  source_ip TEXT,
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
  category_ids JSONB DEFAULT '[]'::jsonb,
  content_text TEXT,
  content_html TEXT,
  content_md TEXT,
  updated_at TIMESTAMPTZ,
  last_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_page_view (
  id TEXT PRIMARY KEY,
  page_id TEXT REFERENCES docs_page(id),
  module_id TEXT REFERENCES docs_module(id),
  version_id TEXT REFERENCES docs_version(id),
  doc_id TEXT,
  module_key TEXT,
  module_name TEXT,
  docs_version TEXT,
  entry_key TEXT,
  title TEXT,
  path TEXT,
  user_id TEXT REFERENCES users(id),
  session_id TEXT,
  read_id TEXT,
  duration_seconds INT,
  scroll_depth NUMERIC,
  viewed_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_favorite (
  id TEXT PRIMARY KEY,
  user_id TEXT REFERENCES users(id),
  module_key TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(user_id, module_key)
);

CREATE TABLE IF NOT EXISTS user_recent_doc (
  id TEXT PRIMARY KEY,
  user_id TEXT REFERENCES users(id),
  doc_id TEXT NOT NULL,
  title TEXT,
  module_key TEXT,
  module_name TEXT,
  docs_version TEXT,
  entry_key TEXT,
  href TEXT,
  viewed_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(user_id, doc_id)
);

CREATE TABLE IF NOT EXISTS docs_search_log (
  id TEXT PRIMARY KEY,
  user_id TEXT,
  ip_address TEXT,
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
  embedding vector(1024),
  embedding_json JSONB,
  metadata_json JSONB,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_docs_embedding_doc_id ON docs_embedding(doc_id);

CREATE TABLE IF NOT EXISTS docs_mcp_log (
  id TEXT PRIMARY KEY,
  tool_name TEXT,
  user_id TEXT,
  query TEXT,
  input_json JSONB,
  result_count INT,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_feedback (
  id TEXT PRIMARY KEY,
  doc_id TEXT NOT NULL,
  page_id TEXT,
  module_key TEXT,
  title TEXT,
  rating TEXT,
  comment TEXT,
  user_id TEXT,
  session_id TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS docs_nav (
  module_key TEXT NOT NULL,
  docs_version TEXT NOT NULL,
  items_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (module_key, docs_version)
);

CREATE TABLE IF NOT EXISTS docs_site_file (
  module_key TEXT NOT NULL,
  docs_version TEXT NOT NULL,
  entry_key TEXT NOT NULL,
  name TEXT NOT NULL,
  content BYTEA NOT NULL,
  content_type TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (module_key, docs_version, entry_key, name)
);

CREATE TABLE IF NOT EXISTS platform_settings (
  key TEXT PRIMARY KEY,
  value_json JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TABLE IF EXISTS store_metadata;
DROP TABLE IF EXISTS modex_store_snapshot;
-- Upgrade databases created by earlier releases. CREATE TABLE IF NOT EXISTS
-- does not add or change columns on an existing installation.
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS roles_json JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE users ADD COLUMN IF NOT EXISTS managed_categories_json JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE users ADD COLUMN IF NOT EXISTS source TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_super_admin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mcp_token TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- Older deployments briefly allowed non-array JSON values in these fields.
-- Keep them normalized so login upserts and user serialization can safely treat
-- them as string arrays.
UPDATE users SET roles_json='[]'::jsonb WHERE jsonb_typeof(roles_json) <> 'array';
UPDATE users SET managed_categories_json='[]'::jsonb WHERE jsonb_typeof(managed_categories_json) <> 'array';

-- Drop the legacy Keycloak group mirror. Team membership (teams + responsible_team)
-- is the only authorization model; SSO groups were never consulted for access.
DROP TABLE IF EXISTS user_groups;
DROP TABLE IF EXISTS groups;
ALTER TABLE users DROP COLUMN IF EXISTS groups_json;

ALTER TABLE teams ADD COLUMN IF NOT EXISTS leaders JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE teams ADD COLUMN IF NOT EXISTS members JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS default_version TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS category_path TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS source_type TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS doc_type TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS mount TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS gitlab_branch TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS gitlab_path TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS deploy_token TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS last_synced_commit TEXT;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMPTZ;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS reads_7d INT NOT NULL DEFAULT 0;
ALTER TABLE docs_module ADD COLUMN IF NOT EXISTS reads_30d INT NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX IF NOT EXISTS docs_module_deploy_token_uidx
  ON docs_module(deploy_token)
  WHERE deploy_token IS NOT NULL AND deploy_token <> '';

ALTER TABLE docs_page ADD COLUMN IF NOT EXISTS category_ids JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE docs_page ADD COLUMN IF NOT EXISTS content_html TEXT;
ALTER TABLE docs_page ADD COLUMN IF NOT EXISTS content_md TEXT;

ALTER TABLE docs_page_view ALTER COLUMN id DROP DEFAULT;
ALTER TABLE docs_page_view ALTER COLUMN id TYPE TEXT USING id::text;
ALTER TABLE docs_page_view ADD COLUMN IF NOT EXISTS doc_id TEXT;
ALTER TABLE docs_page_view ADD COLUMN IF NOT EXISTS module_key TEXT;
ALTER TABLE docs_page_view ADD COLUMN IF NOT EXISTS module_name TEXT;
ALTER TABLE docs_page_view ADD COLUMN IF NOT EXISTS docs_version TEXT;
ALTER TABLE docs_page_view ADD COLUMN IF NOT EXISTS entry_key TEXT;
ALTER TABLE docs_page_view ADD COLUMN IF NOT EXISTS title TEXT;
ALTER TABLE docs_page_view ADD COLUMN IF NOT EXISTS path TEXT;
ALTER TABLE docs_page_view ADD COLUMN IF NOT EXISTS read_id TEXT;
ALTER TABLE docs_search_log ALTER COLUMN id DROP DEFAULT;
ALTER TABLE docs_search_log ALTER COLUMN id TYPE TEXT USING id::text;
ALTER TABLE docs_search_log ADD COLUMN IF NOT EXISTS ip_address TEXT;
ALTER TABLE docs_mcp_log ALTER COLUMN id DROP DEFAULT;
ALTER TABLE docs_mcp_log ALTER COLUMN id TYPE TEXT USING id::text;

-- Remove JSON mirrors from development builds. Rows are reconstructed only
-- from typed columns and foreign-key relationships.
ALTER TABLE users DROP COLUMN IF EXISTS record_json;
ALTER TABLE teams DROP COLUMN IF EXISTS record_json;
ALTER TABLE connected_app DROP COLUMN IF EXISTS record_json;
ALTER TABLE oauth_grant DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_category DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_module DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_version DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_entry DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_release DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_release ADD COLUMN IF NOT EXISTS trigger_type TEXT;
ALTER TABLE docs_release ADD COLUMN IF NOT EXISTS source_ip TEXT;
ALTER TABLE docs_page DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_page_view DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_search_log DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_mcp_log DROP COLUMN IF EXISTS record_json;
ALTER TABLE docs_feedback DROP COLUMN IF EXISTS record_json;
