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
- Backend health: <http://localhost:8080/healthz>
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

Local development defaults to `AUTH_MODE=mock`. For the company Keycloak deployment, create a Keycloak client for Modex and set these values in `deploy/.env` or the production environment:

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
- `GET /api/auth/callback`: exchanges the OAuth2 code, reads userinfo, and creates the Modex session cookie.
- `GET /api/auth/me`: returns the current session user.
- `POST /api/auth/logout`: clears the Modex session cookie.
- `GET /api/config`: returns the frontend-facing auth mode and login URL.

## Deployment Configuration

`deploy/.env.example` keeps deploy-time values out of code. The important groups are:

- Public domains and ports: `APP_BASE_URL`, `FRONTEND_BASE_URL`, `BACKEND_PORT`, `FRONTEND_PORT`, `CORS_ALLOW_ORIGINS`
- PostgreSQL: `DATABASE_URL`, `POSTGRES_*`
- MinIO: `MINIO_ENDPOINT`, `MINIO_PUBLIC_ENDPOINT`, `MINIO_*`
- Meilisearch: `MEILISEARCH_URL`, `MEILISEARCH_PUBLIC_URL`, `MEILI_*`
- Embedding: `EMBEDDING_PROVIDER`, `EMBEDDING_HTTP_URL`, `EMBEDDING_HTTP_API_KEY`, `EMBEDDING_DIM`
- MCP: `MCP_ENABLED`, `MCP_TOKEN`

## docsctl Example

```bash
cd tools/docsctl
go run ./cmd/docsctl validate
DOCS_SOURCE_DIR=../../docs/examples/markdown go run ./cmd/docsctl build
DOCS_SOURCE_DIR=../../docs/examples/markdown go run ./cmd/docsctl package
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

## MCP Example

Start the backend first, then run:

```bash
cd mcp
DOCS_API_BASE_URL=http://localhost:8080 MCP_TOKEN=dev-token go run ./cmd/docs-mcp-server
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

## MVP Notes

The first iteration uses seeded in-memory registry data in the backend so the portal, search, MCP, analytics, and admin-mutation flows are immediately usable. PostgreSQL, MinIO, Meilisearch, pgvector, Keycloak/OIDC, and PostHog integration points are wired through configuration and migrations for the next storage-backed iteration.
