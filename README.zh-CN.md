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

**语言：** [English](README.md) | 中文

Modex 是一个面向团队、企业和开源社区的文档体验平台。它把分散在不同仓库、不同文档框架和不同版本里的工程文档统一接入、发布、检索、阅读和授权，并通过 MCP 让 AI 编程工具可以读取团队的实时文档。

它适合用来建设内部研发文档中心、模块知识库、API/架构手册门户、AI 可检索的工程知识平台，以及需要把 Git 仓库文档自动发布到统一站点的团队级文档系统。

## 核心能力

- **统一文档门户**：Next.js 前端提供首页、分类、文档阅读、个人中心和管理控制台。
- **多来源发布**：`docsctl` 支持 `validate`、`build`、`package`、`deploy`，可接入 Markdown、VuePress、VitePress、Fumadocs 和静态站点。
- **CI 驱动同步**：文档仓库在自己的 CI 中构建并推送标准 artifact 到 Modex，平台侧负责归档、索引和展示。
- **搜索与 AI 问答**：支持关键词、语义、混合搜索；可配置 Chat/Embedding/Rerank 提供商。
- **MCP 访问**：托管部署使用 streamable HTTP MCP server；只支持本地命令的客户端可以继续使用 `npx` stdio wrapper。
- **Skill 包**：Modex Skill 随仓库分发，支持 skill 的客户端可单独安装。
- **权限与组织模型**：支持本地 mock 登录、生产 OIDC/Keycloak、用户、团队、分类责任人、超级管理员和平台级管理权限。
- **发布诊断**：`/api/deploy` 返回阶段化发布结果，方便 CI 排查 artifact 解析、鉴权、资源上传、索引清理和入库问题。
- **运维快照**：`/healthz` 返回 repository、对象存储、搜索/vector、embedding count 和 registry counts。
- **国际化准备**：前端使用 JSON 消息目录，提供一致性检查和 Weblate 接入说明。
- **可测试交付**：Go 单测、前端类型检查、生产构建和 Playwright E2E smoke tests 已接入 CI。

## 仓库结构

```text
modex/
  backend/        Go REST API, auth, analytics, deploy ingest, search, persistence
  frontend/       Next.js portal, admin console, reader, i18n, Playwright tests
  tools/docsctl/  Documentation CLI for validate/build/package/deploy
  mcp/            streamable HTTP MCP server, npx stdio wrapper, client skill
  deploy/         Docker Compose, PostgreSQL/pgvector migration, env templates
  docs/           Operator docs, examples, CI templates, i18n/testing guides
```

## 快速开始

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

本地 Compose 中 MCP 是可选 profile。需要启动 MCP 时：

```bash
cd deploy
docker compose --profile mcp up --build
```

streamable HTTP MCP 地址为：

```text
http://localhost:8787/mcp
```

不使用 Docker 启动应用时，需要先准备 PostgreSQL、Redis、MinIO 和 Meilisearch，然后分别启动后端和前端：

```bash
cd backend
go run ./cmd/modex-api
```

```bash
cd frontend
npm install
npm run dev
```

## 发布一份文档

构建并打包文档：

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

GitLab CI 模板见 [deploy/ci/modex-docs.gitlab-ci.yml](deploy/ci/modex-docs.gitlab-ci.yml)。模板会通过 `MODEX_DOCSCTL_URL` 下载 GitHub Release 里的预编译 `docsctl` 二进制。

## AI 工具接入

托管部署推荐暴露 streamable HTTP MCP endpoint：

```text
https://modex.example.com/mcp
```

MCP server 会把工具调用代理到 Modex 后端。需要鉴权时，为部署设置 `MODEX_API_BASE_URL`，并通过 `MODEX_MCP_TOKEN` 传入用户自己的 MCP token。

如果客户端只支持启动本地 stdio MCP server，可以使用 `npx` wrapper：

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y modex-mcp
```

也可以安装已部署 Modex 后端提供的内网分发包：

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

仓库里的 skill 也可以直接安装：

```bash
npx skills add https://github.com/songkwon/modex/tree/main/mcp/skill
```

更多说明见 [mcp/npx/README.md](mcp/npx/README.md)。

## 发布产物

打 tag 发布时会产出：

- GHCR 镜像：
  - `ghcr.io/songkwon/modex/api`
  - `ghcr.io/songkwon/modex/frontend`
  - `ghcr.io/songkwon/modex/mcp`
- GitHub Release 中的 `docsctl-*` 二进制，覆盖 Linux、macOS 和 Windows。
- GitHub Release 中的 `modex-mcp-*.tgz`，供仍需本地 stdio MCP 的客户端使用。
- Release 文件的 checksums、SBOM、Sigstore bundles 和 build provenance。

`docsctl` 面向文档仓库 CI，优先发二进制；API、前端和托管 MCP server 以容器镜像分发。

## 配置与部署

常用配置文件：

- [deploy/.env.example](deploy/.env.example)：统一环境变量模板，本地开发可直接复制，生产部署请替换所有 secret 和公网 URL。
- [deploy/config.example.yaml](deploy/config.example.yaml)：应用级配置示例，例如 OIDC claim 映射。
- [deploy/docker-compose.yml](deploy/docker-compose.yml)：本地/单机部署编排。

生产部署建议：

- 登录统一走 OIDC/Keycloak，并配置好 `KEYCLOAK_*` 或 `OIDC_*` 环境变量。
- 设置 `COOKIE_SECURE=true`、生产 `COOKIE_DOMAIN` 和精确 CORS origins。
- 替换 PostgreSQL、MinIO、Meilisearch、OIDC、PostHog、cookie、deploy token 等所有 secret。
- 配置真实 Chat/Embedding/Rerank provider 后执行 embedding reindex。
- 如果图表源码不能离开内网，使用内部 Kroki 部署。
- `deploy/.env`、`deploy/config.yaml` 和 `docker compose config` 输出都可能包含 secret，不要公开。

## 测试

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

## 国际化 / Weblate

前端翻译资源位于：

- [frontend/messages/zh-CN.json](frontend/messages/zh-CN.json)
- [frontend/messages/en-US.json](frontend/messages/en-US.json)

`zh-CN` 是源语言。新增文案应先写入 `zh-CN.json`，再同步到其他语言文件，并通过 `useI18n().t(...)` 使用。Weblate 接入说明见 [docs/i18n-weblate.md](docs/i18n-weblate.md)。

## 更多文档

- [项目指南（MDX）](frontend/content/modex-guide.mdx)
- [测试指南](docs/testing.md)
- [国际化与 Weblate](docs/i18n-weblate.md)
- [VuePress 迁移指南](docs/vuepress-migration.md)
- [GitLab CI 模板](deploy/ci/modex-docs.gitlab-ci.yml)
- [生产升级与回滚](docs/operations.md)
- [贡献指南](CONTRIBUTING.md)
- [安全策略](SECURITY.md)

## 许可协议

Modex 使用 [GNU General Public License v3.0](LICENSE) 发布。
