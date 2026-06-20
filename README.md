# Modex

[![GitHub stars](https://img.shields.io/github/stars/songkwon/modex?style=social)](https://github.com/songkwon/modex/stargazers)
[![CI](https://github.com/songkwon/modex/actions/workflows/ci.yml/badge.svg)](https://github.com/songkwon/modex/actions/workflows/ci.yml)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=nextdotjs)](frontend/package.json)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=111)](frontend/package.json)
[![Playwright](https://img.shields.io/badge/E2E-Playwright-45ba4b?logo=playwright)](frontend/playwright.config.ts)
[![MCP](https://img.shields.io/badge/MCP-ready-6f42c1)](mcp/)
[![i18n](https://img.shields.io/badge/i18n-Weblate_ready-2f80ed)](docs/i18n-weblate.md)

**Languages:** [中文](#中文) | [English](#english)

---

## 中文

Modex 是一个面向团队、企业和开源社区的文档体验平台。它把分散在不同仓库、不同文档框架和不同版本里的工程文档统一接入、发布、检索、阅读和授权，并通过 MCP 让 AI 编程工具可以读取团队的实时文档。

它适合用来建设内部研发文档中心、模块知识库、API/架构手册门户、AI 可检索的工程知识平台，以及需要把 Git 仓库文档自动发布到统一站点的团队级文档系统。

### 核心能力

- **统一文档门户**：Next.js 前端提供首页、分类、文档阅读、个人中心和管理控制台。
- **多来源发布**：`docsctl` 支持 `validate`、`build`、`package`、`deploy`，可接入 Markdown、VuePress、VitePress、Fumadocs 和静态站点。
- **CI 驱动同步**：文档仓库在自己的 CI 中构建并推送标准 artifact 到 Modex，平台侧负责归档、索引和展示。
- **搜索与 AI 问答**：支持关键词、语义、混合搜索；可配置 Chat/Embedding/Rerank 提供商。
- **MCP 访问**：内置 stdio MCP server 和 Skill 包，让 Claude Code、Cursor、Windsurf 等工具能搜索和读取 Modex 文档。
- **权限与组织模型**：支持 mock 登录、OIDC/Keycloak、用户、团队、分类责任人、超级管理员和平台级管理权限。
- **发布诊断**：`/api/deploy` 返回阶段化发布结果，方便 CI 排查 artifact 解析、鉴权、资源上传、索引清理和入库问题。
- **运维快照**：`/healthz` 返回 repository、对象存储、搜索/vector、embedding count 和 registry counts。
- **国际化准备**：前端使用 JSON 消息目录，已准备好接入 Weblate。
- **可测试交付**：Go 单测、前端类型检查、生产构建和 Playwright E2E smoke tests 已接入 CI。

### 架构

```text
modex/
  backend/        Go REST API, auth, analytics, deploy ingest, search, persistence
  frontend/       Next.js portal, admin console, reader, i18n, Playwright tests
  tools/docsctl/  Documentation CLI for validate/build/package/deploy
  mcp/            stdio MCP server, npx package, client skill
  deploy/         Docker Compose, PostgreSQL/pgvector migration, env templates
  docs/           Operator docs, examples, CI templates, i18n/testing guides
```

### 快速开始

使用 Docker Compose 启动完整依赖：

```bash
cd deploy
cp .env.example .env
docker compose up --build
```

默认地址：

- 前端：<http://localhost:3456>
- 后端健康检查：<http://localhost:8671/healthz>
- MinIO Console：<http://localhost:9001>
- Meilisearch：<http://localhost:7700>
- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`

不使用 Docker 时，可以分别启动后端和前端：

```bash
cd backend
go run ./cmd/modex-api
```

```bash
cd frontend
npm install
npm run dev
```

### 发布一份文档

构建并打包示例 Markdown 文档：

```bash
cd tools/docsctl
DOCS_SOURCE_DIR=/path/to/docs go run ./cmd/docsctl validate
DOCS_SOURCE_DIR=/path/to/docs go run ./cmd/docsctl build
DOCS_SOURCE_DIR=/path/to/docs go run ./cmd/docsctl package
```

发布到 Modex：

```bash
DOCS_DEPLOY_URL=http://localhost:8671/api/deploy \
DOCS_DEPLOY_TOKEN=your-token \
DOCS_ARTIFACT=/path/to/docs/.modex/docs-artifact.zip \
go run ./cmd/docsctl deploy
```

Deploy Token 可在管理后台为文档源生成。生产 CI 中请把 token 放到 GitLab/GitHub 的 secret variables，不要提交到仓库。

### AI 工具接入

每个用户在 Modex 个人页生成自己的 MCP token，然后把 MCP server 加到 AI 客户端：

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y modex-mcp
```

也可以从已部署的 Modex 后端直接安装内网分发包：

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y https://modex.example.com/api/mcp/dist/modex-mcp.tgz
```

支持 Skill 的客户端可以安装 Modex Skill：

```bash
npx skills add https://modex.example.com
```

更多说明见 [mcp/npx/README.md](mcp/npx/README.md)。

### 配置与部署

常用配置文件：

- [deploy/.env.example](deploy/.env.example)：统一环境变量模板，本地开发可直接复制，生产部署请替换所有 secret 和公网 URL。
- [deploy/config.example.yaml](deploy/config.example.yaml)：应用级配置示例，例如 OIDC claim 映射。
- [deploy/docker-compose.yml](deploy/docker-compose.yml)：本地/单机部署编排。

生产部署建议：

- 登录统一走 OIDC，需配置好 `KEYCLOAK_*` / `OIDC_*` 环境变量。
- 设置 `COOKIE_SECURE=true`、生产 `COOKIE_DOMAIN` 和精确 CORS origins。
- 替换 PostgreSQL、MinIO、Meilisearch、OIDC、PostHog、cookie 等所有 secret。
- 配置真实 Chat/Embedding/Rerank provider 后执行 embedding reindex。
- 如果图表源码不能离开内网，使用内部 Kroki 部署。
- `deploy/.env`、`deploy/config.yaml` 和 `docker compose config` 输出都可能包含 secret，不要公开。

### 测试

```bash
cd backend && go test ./...
cd tools/docsctl && go test ./...
cd mcp && go test ./...

cd frontend
npm run lint
npm run build
npm run e2e
```

详见 [docs/testing.md](docs/testing.md)。

### 国际化 / Weblate

前端翻译资源位于：

- [frontend/messages/zh-CN.json](frontend/messages/zh-CN.json)
- [frontend/messages/en-US.json](frontend/messages/en-US.json)

`zh-CN` 是源语言。新增文案应先写入 `zh-CN.json`，再同步到其他语言文件，并通过 `useI18n().t(...)` 使用。Weblate 接入说明见 [docs/i18n-weblate.md](docs/i18n-weblate.md)。

### 更多文档

- [项目指南（MDX）](frontend/content/modex-guide.mdx)
- [测试指南](docs/testing.md)
- [国际化与 Weblate](docs/i18n-weblate.md)
- [VuePress 迁移指南](docs/vuepress-migration.md)
- [GitLab CI 模板](deploy/ci/modex-docs.gitlab-ci.yml)
- [生产升级与回滚](docs/operations.md)
- [贡献指南](CONTRIBUTING.md)
- [安全策略](SECURITY.md)

### 许可协议

Modex 使用 [GNU General Public License v3.0](LICENSE) 发布。

---

## English

Modex is a documentation experience platform for teams, enterprises, and open-source communities. It brings engineering documentation from many repositories, frameworks, and versions into one governed portal, with publishing, search, reading analytics, permissions, and MCP access for AI coding tools.

Use Modex to build an internal engineering docs hub, module knowledge base, architecture/API handbook portal, AI-searchable documentation platform, or CI-driven documentation publishing system.

### Highlights

- **Unified documentation portal**: a Next.js frontend with home, categories, docs reader, personal workspace, and admin console.
- **Multi-source publishing**: `docsctl` supports `validate`, `build`, `package`, and `deploy` for Markdown, VuePress, VitePress, Fumadocs, and static sites.
- **CI-driven sync**: documentation repositories build in their own CI and push standard artifacts to Modex for archiving, indexing, and rendering.
- **Search and AI answers**: keyword, semantic, and hybrid search with configurable chat, embedding, and rerank providers.
- **MCP access**: bundled stdio MCP server and Skill package so Claude Code, Cursor, Windsurf, and similar tools can search and read live Modex documentation.
- **Permissions and teams**: mock login for local development, OIDC/Keycloak for production, users, teams, category ownership, super admins, and scoped platform management.
- **Deploy diagnostics**: `/api/deploy` returns staged deploy results for artifact parsing, authentication, asset upload, embedding cleanup, and metadata ingest.
- **Operational health**: `/healthz` exposes a lightweight snapshot of repository, object storage, search/vector state, embedding count, and registry counts.
- **Internationalization-ready**: frontend copy uses JSON message catalogs designed for Weblate.
- **Tested delivery**: Go tests, frontend type checks, production build, and Playwright E2E smoke tests are wired into CI.

### Repository Layout

```text
modex/
  backend/        Go REST API, auth, analytics, deploy ingest, search, persistence
  frontend/       Next.js portal, admin console, reader, i18n, Playwright tests
  tools/docsctl/  Documentation CLI for validate/build/package/deploy
  mcp/            stdio MCP server, npx package, client skill
  deploy/         Docker Compose, PostgreSQL/pgvector migration, env templates
  docs/           Operator docs, examples, CI templates, i18n/testing guides
```

### Quick Start

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

Run backend and frontend separately:

```bash
cd backend
go run ./cmd/modex-api
```

```bash
cd frontend
npm install
npm run dev
```

### Publish Documentation

Build and package the Markdown example:

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

### AI Tool Access

Each user generates a personal MCP token in Modex, then adds the MCP server to their AI client:

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

See [mcp/npx/README.md](mcp/npx/README.md) for more client examples.

### Configuration and Deployment

Common configuration files:

- [deploy/.env.example](deploy/.env.example): the unified environment template. Copy it for local development, and replace all secrets and public URLs for production.
- [deploy/config.example.yaml](deploy/config.example.yaml): application-level config such as OIDC claim mapping.
- [deploy/docker-compose.yml](deploy/docker-compose.yml): local and single-node deployment stack.

Production recommendations:

- Login is OIDC-only; configure the `KEYCLOAK_*` / `OIDC_*` environment variables.
- Set `COOKIE_SECURE=true`, a production `COOKIE_DOMAIN`, and exact CORS origins.
- Replace all PostgreSQL, MinIO, Meilisearch, OIDC, PostHog, and cookie secrets.
- Configure real chat, embedding, and rerank providers, then run embedding reindex.
- Use an internal Kroki deployment if diagram source must stay on-prem.
- Treat `deploy/.env`, `deploy/config.yaml`, and `docker compose config` output as secret-bearing material.

### Testing

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

### Internationalization / Weblate

Frontend catalogs:

- [frontend/messages/zh-CN.json](frontend/messages/zh-CN.json)
- [frontend/messages/en-US.json](frontend/messages/en-US.json)

`zh-CN` is the source language. Add new keys there first, mirror them in every locale file, and use `useI18n().t(...)` in components. Weblate setup notes are in [docs/i18n-weblate.md](docs/i18n-weblate.md).

### More Documentation

- [Project Guide (MDX)](frontend/content/modex-guide.mdx)
- [Testing Guide](docs/testing.md)
- [Internationalization and Weblate](docs/i18n-weblate.md)
- [VuePress Migration](docs/vuepress-migration.md)
- [Production upgrades and rollback](docs/operations.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [GitLab CI Template](deploy/ci/modex-docs.gitlab-ci.yml)

### License

Modex is released under the [GNU General Public License v3.0](LICENSE).
