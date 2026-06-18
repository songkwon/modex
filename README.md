# Modex

Modex is an internal Module Documentation Experience platform MVP.

## Structure

- `backend/`: Go REST API with mock registry data, search, model/retrieval settings, analytics placeholders.
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

- Frontend: <http://localhost:3456>
- Backend health: <http://localhost:8671/healthz>
- Redis: `localhost:6379`
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

### Configuration philosophy: Environment variables vs config file

We follow a pragmatic split:

- **Environment variables** — for infrastructure, secrets, and deployment-specific wiring:
  - `AUTH_MODE`, `KEYCLOAK_*`, all `OIDC_*` endpoint / client / redirect settings
  - `COOKIE_*`, `SUPER_ADMIN_USERS`
  - Database, Redis, MinIO, Meilisearch/vector-store, embedding/rerank provider URLs and keys
  - `PORT`, `DATA_DIR`, CORS origins, etc.

- **Application config file (YAML)** — for higher-level, semantic configuration that describes *how the app should interpret data from external systems*. These are good to keep in a version-controlled file (with comments) so changes are reviewable.

  Currently the main candidate is **OIDC user attribute mapping**.

#### OIDC user attribute mapping

Different Keycloak realms and mappers expose user profile data under different claim names.

You can configure the mapping in two ways (they combine with clear precedence):

1. **Recommended for teams**: Put it in a config file (`config.yaml` or similar).
2. **Quick override / CI / one-off**: Use the `OIDC_CLAIM_*` environment variables.

**Precedence (lowest to highest):**
1. Hardcoded defaults in the code (`email`, `picture`, `name`, `department`)
2. Values from the YAML config file
3. Explicit `OIDC_CLAIM_*` environment variables (these always win)

##### Using a config file

Set the `CONFIG_FILE` environment variable, or place the file in one of the conventional locations:

- `config.yaml` / `config.yml` (next to the working directory)
- `configs/config.yaml`
- `/etc/modex/config.yaml`

Example (`deploy/config.example.yaml`):

```yaml
auth:
  user_mapping:
    unique_id_claim: email          # company convention: email is the stable user key
    avatar_claim: picture           # or wxPhotoURL, avatar, etc.
    display_name_claim: name
    secondary_info_claim: department
```

In docker / k8s you typically mount the file:

```yaml
volumes:
  - ./config.yaml:/app/config.yaml:ro
environment:
  CONFIG_FILE: /app/config.yaml
```

##### Environment variable overrides (still supported)

You can continue to (or temporarily) use only environment variables:

```env
OIDC_CLAIM_UNIQUE_ID=email
OIDC_CLAIM_AVATAR=wxPhotoURL
OIDC_CLAIM_DISPLAY_NAME=name
OIDC_CLAIM_SECONDARY_INFO=department
```

These take priority over anything in the config file.

The backend merges claims from **both the ID token and the userinfo endpoint** (ID token is especially useful for custom protocol mappers).

When `unique_id_claim` is set to `email`, the email value becomes the internal `User.ID`. Only use email as the unique key if emails are guaranteed to be stable in your organization.

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
- Redis: `REDIS_URL`, `REDIS_PORT`
- MinIO: `MINIO_ENDPOINT`, `MINIO_PUBLIC_ENDPOINT`, `MINIO_*`
- Meilisearch: `MEILISEARCH_URL`, `MEILISEARCH_PUBLIC_URL`, `MEILI_*`
- Retrieval models: embedding and rerank settings are managed on the admin model page
- MCP: every user generates a personal token on the MCP page and sets it as `MODEX_MCP_TOKEN` in their client

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

## AI Tool Access

Modex ships a user-facing MCP server and Skill package so AI clients (Claude Code,
Cursor, Windsurf, …) can search and read your docs. These packages are for Modex
users, not Modex platform developers.

### 1. MCP via npx

The `mcp/npx` package (`modex-mcp`) is a zero-dependency stdio server. Add it
to your client pointed at a Modex deployment; use the shorter client name
`modex`:

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y modex-mcp
```

See [mcp/npx/README.md](mcp/npx/README.md) for Cursor/Windsurf config.

### 2. MCP from a Modex release

The MCP package is also bundled into Modex releases and served by the backend for
intranet users:

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y https://modex.example.com/api/mcp/dist/modex-mcp.tgz
```

If your deployment keeps MCP in an installable Git package, `npx` can install it
directly:

```bash
npx -y git+https://github.com/your-org/modex-mcp.git
```

### 3. Skill package

For clients that support skills, install the Modex Skill with:

```bash
npx skills add https://modex.example.com
```

If the Skill is maintained in Git:

```bash
npx skills add https://github.com/your-org/modex/tree/main/mcp/skill
```

The Skill carries client-side guidance; MCP is still the runtime channel that
reads Modex data.

### 4. Docker / source for platform operators

The Go MCP server builds alongside the compose stack and runs on demand (stdio,
not a served port):

```bash
cd deploy && docker compose --profile mcp run --rm mcp
```

Local source run:

```bash
cd mcp
MODEX_API_BASE_URL=http://localhost:8671 MODEX_MCP_TOKEN=your-personal-token go run ./cmd/modex-mcp-server
```

Send JSON-RPC lines on stdin:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/list"}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_docs","arguments":{"query":"构建缓存怎么清理","mode":"hybrid","limit":5}}}
```

## Analytics & Admin APIs

Reading statistics use one PostHog project. In `deploy/.env`, configure
`POSTHOG_HOST`, `POSTHOG_PROJECT_API_KEY`, `POSTHOG_PERSONAL_API_KEY`, and
`POSTHOG_PROJECT_ID`. Docker Compose maps the project API key/host into the
Next.js `NEXT_PUBLIC_*` build variables for browser capture, while the backend
uses the same host/project with the personal API key for read-stat queries. No
reading statistics are collected or displayed when PostHog is not configured.
The document eye popover supports 7/30/90-day daily trends and per-reader totals
and average duration.

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
services: PostgreSQL as the registry/source of truth, Redis for sessions/cache/hot
counters and transient jobs, MinIO for document artifacts/site bytes, Meilisearch
for keyword index, and a vector database for embeddings. PostgreSQL + pgvector is
the preferred first vector store; if scale or operations require it, Chroma or
Milvus can be plugged in behind the same vector-store interface.

## GitLab 集成（参考 Mintlify）

我们采用**与 Mintlify 类似的 CI 驱动方式**实现 GitLab 对接：

- **推荐流程**（编译发生在源仓库）：
  1. 在 modex 后台为 Module 配置 `repo_url`、`source_type: "gitlab"`、`gitlab_branch`、`gitlab_path`（仓库内 docs 子目录）。
  2. 生成该 Module 的 **Deploy Token**（通过 admin API `PUT /api/admin/modules/{key}` 设置 `deploy_token`，或未来 UI）。
  3. 在**文档仓库**（例如你的 rd-doc）的 `.gitlab-ci.yml` 中：
     - 运行你的原生构建（`npm run build` for VitePress/VuePress 等）。
     - 使用 `docsctl package`（或 `build` + `package`）生成标准 artifact（包含预构建的 site/ HTML、nav、documents.jsonl 等）。
     - `docsctl deploy`（或直接 curl）把 zip POST 到 modex 的 `/api/deploy`，带 `X-Modex-Deploy-Token` 头。
  4. 推送后自动同步。文档归属到该 Module 关联的“领域”（Category 树中的指定位置）。

- **一个仓库映射到多个位置**（rd-doc 例子）：
  rd-doc 仓库有 `docs/standard/`（规范）、`docs/tools/version-control/` 等。
  你可以用不同 CI job（或 matrix）分别设置 `DOCS_MODULE=rd-standard` + `DOCS_SOURCE_DIR=docs/standard`，
  部署为不同的 Module，然后在 modex 管理后台把它们分配到不同领域（标准规范、工具规范…）。

- **为什么推荐“同步后编译”**：
  - 编译使用仓库自己的环境（正确 Node 版本、依赖、主题、插件）。
  - modex 接收的是**预构建 artifact**（HTML + 静态资源 + 结构化内容 + nav），无需在 modex 里实现各种渲染器。
  - 纯 Markdown 仓库也可以用 `DOCS_BUILDER=markdown` 轻量打包（docsctl 支持）。

- **文档内容与 MinIO**：
  是的。Artifact 中的 `site/`（构建后的 HTML、JS、CSS、图片等静态资源）在生产环境应持久化到 MinIO（当前 MVP 用内存 + snapshot 存储，便于开发和快照恢复；`StorageURI` 字段已预留 `minio://` 路径）。搜索元数据和文本内容进入 store / Meilisearch。

示例 pipeline 见 `docs/pipeline/docs-deploy.example.yml` 和 `tools/docsctl`。

部署鉴权已在 `/api/deploy` 实现，使用管理后台为每个文档源生成的独立 token。

这使得 rd-doc 这样的外部仓库可以持续、结构化地同步到 modex 的指定领域，同时保持构建的完整性。

## License

Modex is licensed under the **GNU Affero General Public License v3.0 or later (AGPL-3.0-or-later)**.

Copyright (C) 2026 songkwon

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU Affero General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option) any
later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE. See the GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License along
with this program. If not, see <https://www.gnu.org/licenses/>.

> AGPL note: if you run a modified version of Modex as a network service, you
> must make the complete source code of your modified version available to its
> users. See the full text in [LICENSE](LICENSE).
