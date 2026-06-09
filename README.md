# Modex

Modex is an internal Module Documentation Experience platform MVP.

## Structure

- `backend/`: Go REST API with mock registry data, search, embedding provider abstraction, analytics placeholders.
- `frontend/`: Next.js portal with home, module cards, info drawer, search, docs reading pages, and admin placeholders.
- `tools/docsctl/`: Go CLI for `validate`, `build`, `package`, and `deploy`.
- `mcp/`: stdio MCP server that calls the backend API.
- `deploy/`: Docker Compose and PostgreSQL migration.
- `docs/`: product and pipeline docs plus sample Markdown source.

## Start Locally

```bash
cd deploy
cp .env.example .env
docker-compose up --build
```

Open:

- Frontend: <http://localhost:3000>
- Backend health: <http://localhost:8671/healthz>
- MinIO console: <http://localhost:9001>
- Meilisearch: <http://localhost:7700>

## Run Without Docker

```bash
cd backend
go run ./cmd/modex-api
```

```bash
cd frontend
npm install
npm run dev
```

## Keycloak / OAuth2 Login

Local development defaults to `AUTH_MODE=mock`. For the company Keycloak deployment, create a Keycloak client for Modex and set these values in `deploy/.env` or the production environment. `AUTH_MODE=keycloak` is accepted as an alias of `oidc`.

```env
AUTH_MODE=oidc
APP_BASE_URL=https://modex-api.example.com
FRONTEND_BASE_URL=https://modex.example.com
NEXT_PUBLIC_API_BASE_URL=https://modex-api.example.com
CORS_ALLOW_ORIGINS=https://modex.example.com

KEYCLOAK_BASE_URL=https://keycloak.example.com
KEYCLOAK_REALM=your-realm
OIDC_CLIENT_ID=modex
OIDC_CLIENT_SECRET=replace-with-keycloak-secret
OIDC_REDIRECT_URL=https://modex-api.example.com/api/auth/callback
OIDC_SCOPES=openid profile email

COOKIE_DOMAIN=.example.com
COOKIE_SAME_SITE=lax
COOKIE_SECURE=true
```

Keycloak client settings:

- Valid redirect URI: `https://modex-api.example.com/api/auth/callback`
- Web origin: `https://modex.example.com`
- Access type: confidential, if using `OIDC_CLIENT_SECRET`

If your Keycloak endpoints are non-standard, override `OIDC_ISSUER_URL` or the explicit endpoint variables:

- `OIDC_AUTH_URL`
- `OIDC_TOKEN_URL`
- `OIDC_USERINFO_URL`
- `OIDC_END_SESSION_URL`

The backend exposes:

- `GET /api/auth/login`: redirects to Keycloak.
- `GET /api/auth/callback`: exchanges the OAuth2 code, reads userinfo, creates the Modex session cookie, and syncs the user + groups into the directory. On failure it redirects to the portal with `?login_error=...` and logs the detail server-side.
- `GET /api/auth/me`: returns the current session user (401 when not logged in).
- `POST /api/auth/logout`: clears the Modex session cookie.
- `GET /api/config`: returns the frontend-facing auth mode and login URL (empty when OIDC is not fully configured).

### Login model

Login is a real, cookie-backed action in **both** modes — the backend no longer
silently impersonates a seeded user. Anonymous visitors can still browse the
portal; `GET /api/auth/me` simply returns 401 until they log in.

- `AUTH_MODE=oidc` (or `keycloak`): the portal "登录" button sends the browser to
  `/api/auth/login` → Keycloak → `/api/auth/callback`, which sets the session cookie.
- `AUTH_MODE=mock` (local dev): `POST /api/auth/mock-login` creates a real session
  cookie. Pass `{"username":"alice"}` to log in as a specific seeded user.

### Why Keycloak login can fail

If `AUTH_MODE=oidc` is set but login does not work, check, in order:

1. **`GET /api/config` shows `oidc_login_enabled: false`** → the issuer/endpoints or
   `OIDC_CLIENT_ID` are missing. Set `KEYCLOAK_BASE_URL` + `KEYCLOAK_REALM` (or
   `OIDC_ISSUER_URL`) and `OIDC_CLIENT_ID`.
2. **Redirect URI mismatch** → `OIDC_REDIRECT_URL` must exactly equal the value
   registered in the Keycloak client (`{APP_BASE_URL}/api/auth/callback`).
3. **`login_url` points at the wrong host** → set `APP_BASE_URL` to the URL the
   browser uses to reach the backend (e.g. `http://localhost:8671` locally).
4. **CORS / cookies** → `CORS_ALLOW_ORIGINS` must include the frontend origin; for
   cross-subdomain prod set `COOKIE_DOMAIN=.example.com`, `COOKIE_SECURE=true`.
5. Otherwise read the backend log — callback errors (including provider
   `error_description`) are logged and echoed to the portal via `?login_error=`.

## User, Group & Permission Management

The directory is managed from `/admin/users` (super-admin only):

- `GET /api/admin/users` (optional `?keyword=`), `POST /api/admin/users`
- `GET|PUT|DELETE /api/admin/users/{id}`
- `GET /api/admin/groups`, `POST /api/admin/groups`

OIDC logins upsert the user into this directory and refresh their groups and
last-login timestamp. Groups referenced by a user are auto-registered.

### Roles & platform permissions

- **Super admin**: configured via `SUPER_ADMIN_USERS` (comma-separated
  usernames/emails). Matched on login (mock or OIDC), granted the `admin` role,
  and may manage every platform plus users/permissions. `GET /api/auth/me`
  reports `is_super_admin`.
- **Platform admin**: a user with the `admin` role and `managed_categories`
  set to the platform (category) IDs they govern. A managed platform covers its
  sub-platforms (e.g. `engineering` covers `engineering.cbb`).

Platform-scoped writes are enforced server-side:

- Create/update/delete categories, modules, versions, entries require manage
  rights on the relevant platform (super admins bypass).
- Top-level platform creation and all user/group/permission management are
  super-admin only.
- Search/embedding reindex requires an admin session.
- `POST /api/admin/modules/{key}/migrate` reassigns a module to other
  platform(s) and requires manage rights on **both** source and target
  (super admins bypass).

## AI Search

The home page has a centered search (ChatGPT-style). As you type it shows live
results with an entry-type icon, platform breadcrumb, title, a context snippet,
and highlighted matched keywords. The sparkle **询问 AI** action calls
`POST /api/ask`, which retrieves the top documents and either forwards them to an
external LLM (`ASK_HTTP_URL`) or returns an extractive answer with cited sources.

## Deployment Configuration

`deploy/.env.example` keeps deploy-time values out of code. The important groups are:

- Public domains and ports: `APP_BASE_URL`, `FRONTEND_BASE_URL`, `BACKEND_PORT`, `FRONTEND_PORT`, `CORS_ALLOW_ORIGINS`
- Frontend API routing: `NEXT_PUBLIC_API_BASE_URL` is used by browser-side requests; `INTERNAL_API_BASE_URL` is used by Next.js server rendering inside Docker. In Compose, keep it as `http://backend:8671` unless you change `BACKEND_PORT`.
- PostgreSQL: `DATABASE_URL`, `POSTGRES_*`
- MinIO: `MINIO_ENDPOINT`, `MINIO_PUBLIC_ENDPOINT`, `MINIO_*`
- Meilisearch: `MEILISEARCH_URL`, `MEILISEARCH_PUBLIC_URL`, `MEILI_*`
- Embedding: `EMBEDDING_PROVIDER`, `EMBEDDING_HTTP_URL`, `EMBEDDING_HTTP_API_KEY`, `EMBEDDING_DIM`
- MCP: `MCP_ENABLED`, `MCP_TOKEN`

## docsctl Examples

```bash
cd tools/docsctl
DOCS_SOURCE_DIR=/path/to/vuepress-project go run ./cmd/docsctl init
DOCS_SOURCE_DIR=/path/to/wiki-root go run ./cmd/docsctl discover
DOCS_SOURCE_DIR=/path/to/wiki-root DOCS_DISCOVER_WRITE=true go run ./cmd/docsctl discover
go run ./cmd/docsctl validate
DOCS_SOURCE_DIR=../../docs/examples/markdown go run ./cmd/docsctl build
DOCS_SOURCE_DIR=../../docs/examples/markdown go run ./cmd/docsctl package
DOCS_DEPLOY_URL=http://localhost:8671/api/deploy DOCS_ARTIFACT=../../docs/examples/markdown/.modex/docs-artifact.zip go run ./cmd/docsctl deploy
```

Additional examples:

```bash
DOCS_SOURCE_DIR=../../docs/examples/vuepress go run ./cmd/docsctl package
DOCS_SOURCE_DIR=../../docs/examples/fumadocs go run ./cmd/docsctl package
```

The artifact is written to `docs/examples/markdown/.modex/docs-artifact.zip` and includes:

- `site/`
- `manifest.json`
- `metadata.json`
- `nav.json`
- `documents.jsonl`
- `llms.txt`
- optional `llms-full.txt`
- `assets/`

The VuePress and Fumadocs examples use small local build scripts so the packaging path can be tested without installing full framework dependencies. Real projects should replace those scripts with commands such as `pnpm docs:build`, `npm run build`, or the team-standard build command.

For existing RD/VuePress sites, see `docs/vuepress-migration.md`.

`docsctl discover` recursively scans existing documentation roots, detects
VuePress, Fumadocs, static HTML, and Markdown projects, and can create
`docs.yaml` in place when `DOCS_DISCOVER_WRITE=true` is set. Use
`DOCS_DISCOVER_DEPTH` to control traversal depth.

## MCP Example

Start the backend first, then run:

```bash
cd mcp
DOCS_API_BASE_URL=http://localhost:8671 MCP_TOKEN=dev-token go run ./cmd/docs-mcp-server
```

Send JSON-RPC lines on stdin:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/list"}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_docs","arguments":{"query":"构建缓存怎么清理","mode":"hybrid","limit":5}}}
```

## Analytics & Admin APIs

Reading statistics are tracked in the backend and surfaced in the admin portal:

- `POST /api/analytics/page-view`: records a page view (`doc_id`, `session_id`).
- `POST /api/analytics/read-progress`: updates dwell time and scroll depth.
- `GET /api/admin/analytics/pages`: aggregated PV / UV / 7-day / 30-day reads per page.

The docs reading page records views automatically via the `PageViewTracker`
client component, and `frontend/lib/analytics.ts` holds the PostHog init and
`capture()` event helpers (enabled by setting `NEXT_PUBLIC_POSTHOG_KEY`).

Admin registry mutations are implemented against the in-memory store:

- Categories: `POST /api/admin/categories`, `PUT|DELETE /api/admin/categories/{id}`
- Modules: `POST /api/admin/modules`, `PUT /api/admin/modules/{module_key}`
- Versions: `POST /api/admin/modules/{module_key}/versions`, `PUT .../versions/{docs_version}`
- Entries: `POST .../versions/{docs_version}/entries`, `PUT|DELETE /api/admin/entries/{entry_id}`
- Releases: `GET /api/admin/releases/{release_id}`, `POST /api/admin/releases/{release_id}/rollback`

## Search Index Maintenance

Semantic and hybrid search use embeddings produced by the configured provider
and cached per document. Rebuild the cache after publishing new content:

- `POST /api/embeddings/reindex`: (re)embeds every page and returns the count.
- `POST /api/search/reindex`: rebuilds the index and reports document counts.

Both are also available from the **索引维护** panel on `/admin`. Embeddings are
otherwise computed lazily on first semantic/hybrid query and cached; keyword-only
search never calls the embedding provider. Publishing a new artifact invalidates
the cached vectors for that module/version automatically.

## Durable Store Snapshots

Set `DATA_DIR` to make the registry survive restarts. On boot the backend loads
`${DATA_DIR}/modex-store.json` (or seeds fresh data when absent), saves
periodically (`DATA_SAVE_INTERVAL_SECONDS`, default 60), and writes a final
snapshot on graceful shutdown (SIGINT/SIGTERM). Writes are atomic (temp file +
rename). In Docker Compose this is a `backend-data` named volume mounted at
`/data`, so `docker compose restart` keeps modules, users, analytics, and search
indexes. Leave `DATA_DIR` empty for a pure in-memory store.

## Branding

The Modex mark lives at `frontend/app/icon.svg` (auto-served favicon),
`frontend/app/apple-icon.png`, and `frontend/public/logo.svg` (header). PNG icon
sizes and the web manifest are generated from the SVG; regenerate with
`rsvg-convert -w <size> -h <size> app/icon.svg -o <out>.png`.

## MVP Notes

The full product loop is functional end-to-end: `docsctl deploy` uploads an
artifact to `POST /api/deploy`, which parses the zip and ingests modules,
versions, entries, pages, nav, built HTML, and site assets so they immediately
appear in the portal, search, and MCP. Mock + Keycloak/OIDC login, user/group
management, analytics, admin CRUD, real search/embedding reindexing, and durable
snapshot persistence are all implemented.

The next infrastructure iteration replaces the snapshot store with managed
services: PostgreSQL (source of truth), MinIO (artifact/site bytes), Meilisearch
(keyword index), and pgvector (embeddings). Configuration, the `001_init.sql`
migration, and the provider seams are already in place for that work.
