# Modex

[![GitHub stars](https://img.shields.io/github/stars/songkwon/modex?style=social)](https://github.com/songkwon/modex/stargazers)
[![CI](https://github.com/songkwon/modex/actions/workflows/ci.yml/badge.svg)](https://github.com/songkwon/modex/actions/workflows/ci.yml)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=nextdotjs)](frontend/package.json)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=111)](frontend/package.json)
[![Playwright](https://img.shields.io/badge/E2E-Playwright-45ba4b?logo=playwright)](frontend/playwright.config.ts)
[![MCP](https://img.shields.io/badge/MCP-streamable_HTTP-6f42c1)](mcp/)
[![i18n](https://img.shields.io/badge/i18n-ready-2f80ed)](docs/i18n-weblate.md)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/songkwon/modex)

**Language:** English | [中文](README.zh-CN.md)

Modex is a documentation experience platform for teams, enterprises, and open-source communities. It brings engineering documentation from many repositories, frameworks, and versions into one governed portal, with publishing, search, reading analytics, permissions, and MCP access for AI coding tools.

Use Modex to build an internal engineering docs hub, module knowledge base, architecture/API handbook portal, AI-searchable documentation platform, or CI-driven documentation publishing system.

## Highlights

- **Unified documentation portal**: a Next.js frontend with home, categories, docs reader, personal workspace, and admin console.
- **Multi-source publishing**: `docsctl` supports `validate`, `build`, `package`, and `deploy` for Markdown, VuePress, VitePress, Fumadocs, and static sites.
- **CI-driven sync**: documentation repositories build in their own CI and push standard artifacts to Modex for archiving, indexing, and rendering.
- **Search and AI answers**: keyword, semantic, and hybrid search with configurable chat, embedding, and rerank providers.
- **MCP access**: a streamable HTTP MCP server for hosted deployments, plus an `npx` stdio wrapper for clients that only support local MCP commands.
- **Skill package**: a Modex Skill is shipped in the repository and can be installed separately by clients that support skills.
- **Permissions and teams**: mock login for local development, OIDC/Keycloak for production, users, teams, category ownership, super admins, and scoped platform management.
- **Deploy diagnostics**: `/api/deploy` returns staged deploy results for artifact parsing, authentication, asset upload, embedding cleanup, and metadata ingest.
- **Operational health**: `/healthz` exposes a lightweight snapshot of repository, object storage, search/vector state, embedding count, and registry counts.
- **Internationalization-ready**: frontend copy uses JSON message catalogs, consistency checks, and Weblate setup notes.
- **Tested delivery**: Go tests, frontend type checks, production build, and Playwright E2E smoke tests are wired into CI.

## Repository Layout

```text
modex/
  backend/        Go REST API, auth, analytics, deploy ingest, search, persistence
  frontend/       Next.js portal, admin console, reader, i18n, Playwright tests
  tools/docsctl/  Documentation CLI for validate/build/package/deploy
  mcp/            streamable HTTP MCP server, npx stdio wrapper, client skill
  deploy/         Docker Compose, PostgreSQL/pgvector migration, env templates
  docs/           Operator docs, examples, CI templates, i18n/testing guides
```

## Quick Start

Start the full local stack with Docker Compose:

```bash
cd deploy
cp .env.example .env
docker compose up --build
```

Default endpoints:

- Frontend: <http://localhost:3456>
- Backend health: <http://localhost:8671/healthz>
- MinIO Console: <http://localhost:9001>
- Meilisearch: <http://localhost:7700>
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

The MCP server is optional in local Compose. Enable it with the `mcp` profile:

```bash
cd deploy
docker compose --profile mcp up --build
```

Then point streamable HTTP MCP clients at:

```text
http://localhost:8787/mcp
```

To run backend and frontend separately, make sure PostgreSQL, Redis, MinIO, and Meilisearch are available, then start:

```bash
cd backend
go run ./cmd/modex-api
```

```bash
cd frontend
npm install
npm run dev
```

## Publish Documentation

Build and package documentation:

```bash
cd tools/docsctl
DOCS_SOURCE_DIR=/path/to/docs go run ./cmd/docsctl validate
DOCS_SOURCE_DIR=/path/to/docs go run ./cmd/docsctl build
DOCS_SOURCE_DIR=/path/to/docs go run ./cmd/docsctl package
```

Deploy it to Modex:

```bash
DOCS_DEPLOY_URL=http://localhost:8671/api/deploy \
DOCS_DEPLOY_TOKEN=your-token \
DOCS_ARTIFACT=/path/to/docs/.modex/docs-artifact.zip \
go run ./cmd/docsctl deploy
```

Generate the Deploy Token from the Modex admin console. In production CI, store it in GitLab/GitHub secret variables and never commit it.

For GitLab CI, see [deploy/ci/modex-docs.gitlab-ci.yml](deploy/ci/modex-docs.gitlab-ci.yml). The template downloads a prebuilt `docsctl` binary from GitHub Releases through `MODEX_DOCSCTL_URL`.

## AI Tool Access

For hosted deployments, run the MCP image and expose the streamable HTTP endpoint:

```text
https://modex.example.com/mcp
```

The MCP server proxies tool calls to the Modex backend. Set `MODEX_API_BASE_URL` to the backend URL and pass a user MCP token with `MODEX_MCP_TOKEN` when the deployment needs authenticated access.

For clients that only support local stdio MCP servers, use the `npx` wrapper:

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y modex-mcp
```

You can also install the package served by a deployed Modex backend:

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y https://modex.example.com/api/mcp/dist/modex-mcp.tgz
```

For clients that support skills:

```bash
npx skills add https://modex.example.com
```

The repository-hosted skill can also be installed from [mcp/skill](mcp/skill):

```bash
npx skills add https://github.com/songkwon/modex/tree/main/mcp/skill
```

See [mcp/npx/README.md](mcp/npx/README.md) for more client examples.

## Release Artifacts

Tagged releases publish:

- GHCR images for the API, frontend, and MCP server:
  - `ghcr.io/songkwon/modex/api`
  - `ghcr.io/songkwon/modex/frontend`
  - `ghcr.io/songkwon/modex/mcp`
- `docsctl-*` binaries in GitHub Releases for Linux, macOS, and Windows.
- `modex-mcp-*.tgz` in GitHub Releases for local stdio MCP clients.
- Checksums, SBOM, Sigstore bundles, and build provenance for release files.

`docsctl` is intentionally distributed as a binary for CI jobs. The API, frontend, and hosted MCP server are distributed as container images.

## Configuration and Deployment

Common configuration files:

- [deploy/.env.example](deploy/.env.example): the unified environment template. Copy it for local development, and replace all secrets and public URLs for production.
- [deploy/config.example.yaml](deploy/config.example.yaml): application-level config such as OIDC claim mapping.
- [deploy/docker-compose.yml](deploy/docker-compose.yml): local and single-node deployment stack.

Production recommendations:

- Use OIDC/Keycloak for login and configure the `KEYCLOAK_*` or `OIDC_*` environment variables.
- Set `COOKIE_SECURE=true`, a production `COOKIE_DOMAIN`, and exact CORS origins.
- Replace all PostgreSQL, MinIO, Meilisearch, OIDC, PostHog, cookie, and deploy-token secrets.
- Configure real chat, embedding, and rerank providers, then run embedding reindex.
- Use an internal Kroki deployment if diagram source must stay on-prem.
- Treat `deploy/.env`, `deploy/config.yaml`, and `docker compose config` output as secret-bearing material.

## Testing

```bash
cd backend && go test ./...
cd tools/docsctl && go test ./...
cd mcp && go test ./...

cd frontend
npm run lint
npm run build
npm run e2e
```

See [docs/testing.md](docs/testing.md).

## Internationalization / Weblate

Frontend catalogs:

- [frontend/messages/zh-CN.json](frontend/messages/zh-CN.json)
- [frontend/messages/en-US.json](frontend/messages/en-US.json)

`zh-CN` is the source language. Add new keys there first, mirror them in every locale file, and use `useI18n().t(...)` in components. Weblate setup notes are in [docs/i18n-weblate.md](docs/i18n-weblate.md).

## More Documentation

- [Project Guide (MDX)](frontend/content/modex-guide.mdx)
- [Testing Guide](docs/testing.md)
- [Internationalization and Weblate](docs/i18n-weblate.md)
- [VuePress Migration](docs/vuepress-migration.md)
- [GitLab CI Template](deploy/ci/modex-docs.gitlab-ci.yml)
- [Production upgrades and rollback](docs/operations.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)

## License

Modex is released under the [GNU General Public License v3.0](LICENSE).
