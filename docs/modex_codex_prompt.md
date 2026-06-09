# Modex 研发文档平台 Codex 开发任务说明

> 项目暂定名：**Modex**  
> 定位：Module Documentation Experience，面向公司内部多模块、多技术栈、多仓库的研发文档平台。  
> 目标：先实现一个可运行的 MVP，然后逐步扩展成完整的研发文档门户、搜索和 AI/MCP 文档访问平台。

---

## 1. 项目背景

公司存在多个研发领域和技术栈，包括：

- NC
- CAD
- KDC
- 应用
- 工程化
- Delphi
- C++
- Java
- Go
- 前端
- 测试
- 运维

现有 CBB 平台主要负责 Delphi / C++ 模块构建，不适合作为全公司的文档平台底座。  
因此需要独立建设一个研发文档平台，用于统一管理各模块文档的：

- 发布
- 展示
- 分类
- 版本
- 搜索
- 语义搜索
- MCP / AI 读取
- 阅读统计
- SSO 登录
- 后续多种文档 builder 扩展

第一阶段不是做一个大而全的知识平台，而是做一个具备核心闭环的 MVP。

---

## 2. 核心设计原则

1. 文档平台独立于 CBB。
2. CBB 只是文档发布来源之一。
3. 文档内容随代码仓库维护。
4. 文档平台负责展示、搜索、统计和治理。
5. 不强制统一所有文档框架。
6. 强制统一发布产物协议。
7. GitLab 项目通过公共 Pipeline 自助发布。
8. SVN / 老项目后续通过中心 Builder Server 拉代码构建发布。
9. 第一阶段支持 Markdown、Static HTML、VuePress。
10. 第一阶段支持关键词搜索、语义搜索和 Hybrid Search。
11. 第一阶段支持 MCP Server。
12. 门户不直接扫描对象存储，而是以 PostgreSQL 中的 Docs Registry 为唯一事实源。
13. docs.yaml 只声明文档入口，不重复声明模块、版本、Owner、分类、权限。
14. cbb.toml 作为模块工程元数据来源。
15. Registry 负责平台治理字段。

---

## 3. 技术栈

### 3.1 后端

- Go
- REST API
- PostgreSQL
- MinIO
- Meilisearch
- pgvector 可选，如果实现方便则使用 PostgreSQL + pgvector 存储 embedding
- OIDC SSO 预留，MVP 可以先实现 mock login
- PostHog 预留
- MCP Server，Go 实现
- Docker Compose

### 3.2 前端

- Next.js
- TypeScript
- shadcn/ui
- Tailwind CSS
- TanStack Query
- TanStack Table 可选
- PostHog JS SDK 预留

### 3.3 工具

- docsctl，Go CLI
- 支持 markdown builder
- 支持 static html importer
- 支持 vuepress builder

---

## 4. 仓库结构

请创建 monorepo：

```text
modex/
  backend/
  frontend/
  deploy/
  docs/
  tools/
    docsctl/
  mcp/
```

说明：

- `backend/`：Go 后端 API。
- `frontend/`：Next.js 前端门户。
- `deploy/`：docker-compose、数据库初始化、部署配置。
- `docs/`：项目说明文档。
- `tools/docsctl/`：文档构建、打包、发布 CLI。
- `mcp/`：MCP Server，可以是单独 Go module，也可以共享 backend 代码。

---

## 5. 总体架构

```text
GitLab 项目
  └─ include 公共 docs pipeline
      └─ docsctl validate/build/package/deploy
          └─ docs-deploy-api
              ├─ PostgreSQL: Docs Registry
              ├─ MinIO: HTML / 静态资源 / 文档包
              ├─ Meilisearch: 关键词搜索 / Facet
              ├─ Vector Store: 语义搜索
              ├─ Docs Portal: 前端门户
              └─ Docs MCP Server: AI 读取入口

SVN / 老项目，后续阶段
  └─ docs-builder-server
      └─ 使用只读 SVN 凭据拉取代码
          └─ docsctl build/package/deploy
              └─ docs-deploy-api
```

---

## 6. 配置文件边界

### 6.1 cbb.toml

如果项目中存在 `cbb.toml`，它作为模块工程元数据来源。

示例：

```toml
[package]
name = "DemoModule"
version = "1.2.3"
channel = "default"
description = "示例模块"
authors = ["alice"]
edition = "2025"
keywords = ["demo", "cad"]
```

映射关系：

| cbb.toml 字段 | 文档平台字段 |
|---|---|
| package.name | module_key / module_name |
| package.version | package_version |
| package.channel | channel |
| package.description | description |
| package.authors | maintainers |
| package.edition | edition |
| package.keywords | keywords / tags |

注意：

- `package.version` 是工程包版本。
- 文档平台自己的版本叫 `docs_version`，例如 `latest`、`main`、`v1.2`、`legacy`。
- 不要把 `package.version` 和 `docs_version` 混为一个字段。
- `authors` 可以作为 maintainers 候选，但不要直接等同平台 Owner。
- `keywords` 可以作为搜索标签，但不要自动决定平台分类。

---

### 6.2 docs.yaml

`docs.yaml` 只声明文档入口，不重复声明模块名、版本、Owner、分类、权限。

最小示例：

```yaml
entries:
  - key: guide
    title: 模块落地指导
    type: markdown
    source: docs/integration-guide.md

  - key: maintenance
    title: 模块维护说明
    type: markdown
    source: docs/maintenance-guide.md
```

静态 HTML 老文档示例：

```yaml
entries:
  - key: legacy
    title: 历史文档
    type: static
    source: legacy/docomatic/html
```

VuePress 文档示例：

```yaml
entries:
  - key: guide
    title: VuePress 使用说明
    type: vuepress
    source: docs
    build: pnpm docs:build
    output: docs/.vuepress/dist
```

后续可扩展示例：

```yaml
entries:
  - key: api
    title: 接口文档
    type: openapi
    source: docs/api/openapi.yaml

  - key: reference
    title: C++ API Reference
    type: doxygen
    source: Doxyfile
    output: build/doxygen/html
```

---

### 6.3 Registry

平台 Registry 负责治理字段，不让项目仓库自己随意控制。

Registry 管：

- 分类
- 路径
- Owner
- 权限
- 默认版本
- 模块展示信息
- 发布权限
- 阅读权限
- 旧路径 redirect，后续阶段做

---

## 7. 元数据合并规则

`docsctl` 需要支持读取：

1. CI 环境变量
2. cbb.toml
3. docs.yaml
4. Registry 默认配置

优先级：

```text
CI 变量 > cbb.toml > docs.yaml > Registry
```

例如：

```bash
DOCS_MODULE=DemoModule DOCS_VERSION=latest docsctl build
```

生成的 `metadata.json` 应该类似：

```json
{
  "module_key": "DemoModule",
  "module_name": "DemoModule",
  "docs_version": "latest",
  "package_version": "1.2.3",
  "channel": "default",
  "description": "示例模块",
  "authors": ["alice"],
  "edition": "2025",
  "keywords": ["demo", "cad"],
  "source": {
    "metadata_file": "cbb.toml"
  }
}
```

---

## 8. 标准文档包

`docsctl package` 最终生成标准文档包：

```text
docs-artifact.zip
  ├─ site/
  │   └─ index.html
  ├─ manifest.json
  ├─ metadata.json
  ├─ nav.json
  ├─ documents.jsonl
  ├─ embeddings.jsonl        # 可选
  ├─ llms.txt
  ├─ llms-full.txt           # 可选
  └─ assets/
```

说明：

- `site/`：给用户阅读的 HTML 静态站。
- `manifest.json`：描述文档包包含哪些 entry。
- `metadata.json`：模块、版本、发布来源等元数据。
- `nav.json`：文档目录。
- `documents.jsonl`：关键词搜索、语义搜索和 MCP 精细检索使用。
- `embeddings.jsonl`：如果在 docsctl 阶段生成 embedding，则放这里；MVP 可不生成。
- `llms.txt`：给 LLM 快速理解当前模块文档结构的入口摘要。
- `llms-full.txt`：可选，将所有正文聚合为 AI 友好的完整文本；大文档可以跳过。
- `assets/`：图片、图表、附件。

平台发布的主展示产物是 HTML，但不能只发布 HTML。

---

## 9. llms.txt 规范

`llms.txt` 必须生成。

它不替代 `documents.jsonl`。

两者定位：

| 文件 | 用途 |
|---|---|
| documents.jsonl | 搜索、语义检索、MCP 精细召回 |
| llms.txt | LLM 快速理解模块文档结构和重要入口 |
| llms-full.txt | 可选，聚合完整正文，适合小型文档站 |

`llms.txt` 建议内容：

```text
# DemoModule

Description: 示例模块
Docs Version: latest
Package Version: 1.2.3
Channel: default
Keywords: demo, cad

## Entries

- 模块落地指导: /guide
  Type: markdown
  Source: docs/integration-guide.md
  Summary: 面向业务开发人员的模块接入、部署、接口和异常处理说明。

- 模块维护说明: /maintenance
  Type: markdown
  Source: docs/maintenance-guide.md
  Summary: 面向维护开发人员的架构、设计、流程和维护说明。

## Recommended Reading

1. 模块落地指导
2. 模块维护说明

## Notes for AI

Use documents.jsonl for precise retrieval.
Use this file only as a high-level map of the documentation.
```

---

## 10. 第一阶段支持的文档类型

MVP 支持：

1. Markdown
2. Static HTML import
3. VuePress

暂时不做：

- OpenAPI builder
- Doxygen builder
- Javadoc builder
- Fumadocs builder
- Docusaurus builder
- SVN builder-server

---

## 11. 文档编写规范

每个新模块推荐维护两份 Markdown：

```text
docs/
  integration-guide.md
  maintenance-guide.md
  assets/
  diagrams/
docs.yaml
```

### 11.1 integration-guide.md

模块落地指导，面向业务开发人员。

建议章节：

1. 模块概述
2. 功能边界
3. 接口设计
4. 部署与运行
5. 异常与错误处理
6. 已知风险与影响面

### 11.2 maintenance-guide.md

模块维护说明，面向维护开发人员。

建议章节：

1. 模块概述
2. 功能边界
3. 总体架构
4. 核心设计思路与设计原则
5. 模块结构
6. 核心流程与时序逻辑
7. 前后端设计
8. 质量与可维护性

---

## 12. 门户展示设计

### 12.1 首页

首页包含：

- 顶部全局搜索框
- 层级分类树 / 分类 Tab
- 模块卡片
- 最近更新
- 热门文档
- 我的关注
- 管理入口
- 用户头像

分类支持层级，例如：

```text
NC
  - NC 基础平台
  - NC 加工
  - NC 后处理

CAD
  - CAD 内核
  - CAD 插件
  - 图形渲染

KDC
  - KDC 平台
  - KDC 数据服务

应用
  - PMS
  - 设备联网
  - 订单服务

工程化
  - CBB
  - CI/CD
  - Review Board
  - SonarQube
```

### 12.2 模块卡片

模块卡片展示：

- 模块名称
- 分类
- 默认版本
- 最近更新
- 状态
- 标签 / keywords
- Info 按钮

示例：

```text
DemoModule                         ⓘ
CAD / 示例模块
默认版本：latest
工程版本：1.2.3
最近更新：2026-06-09
标签：demo / cad
```

交互：

- 点击卡片主体：进入默认版本文档。
- 点击 Info：打开模块信息抽屉。
- 点击收藏：加入我的关注。

### 12.3 模块 Info 抽屉

展示：

- 模块名称
- 描述
- 分类
- Owner
- maintainers / authors
- 来源仓库
- 默认文档版本
- package_version
- channel
- edition
- keywords
- 可用版本
- 最近发布
- 阅读量
- 发布记录入口
- 查看源码入口

### 12.4 文档阅读页

布局：

```text
顶部：
  面包屑 / 模块名 / 版本选择 / 搜索本模块 / 查看源码 / 提交反馈

左侧：
  当前 Entry 文档目录

中间：
  文档正文

右侧：
  本文目录
  文档元数据
  阅读统计
```

---

## 13. 搜索设计

MVP 必须支持：

1. 关键词搜索
2. 条件过滤
3. Facet
4. 分页
5. 排序
6. 语义搜索
7. Hybrid Search

### 13.1 搜索引擎

第一阶段使用：

- Meilisearch：关键词搜索、Facet、过滤
- Vector Store：语义搜索

Vector Store 可选实现：

1. PostgreSQL + pgvector，推荐。
2. 如果 pgvector 接入复杂，可以先用 PostgreSQL JSONB 存向量 + 简单 cosine 计算作为 MVP。
3. 不要把向量搜索逻辑写死，抽象为 `EmbeddingStore` 接口。

### 13.2 Embedding Provider

实现可插拔 embedding provider：

```text
EmbeddingProvider
  - Name()
  - EmbedText(ctx, text) ([]float32, error)
  - EmbedBatch(ctx, texts) ([][]float32, error)
```

第一阶段至少支持：

1. `MockEmbeddingProvider`：本地 deterministic embedding，用于开发测试。
2. `HTTPEmbeddingProvider`：调用外部 embedding 服务，配置 URL 和 API Key。

配置示例：

```env
EMBEDDING_PROVIDER=mock
EMBEDDING_HTTP_URL=
EMBEDDING_HTTP_API_KEY=
EMBEDDING_DIM=384
```

### 13.3 搜索模式

`POST /api/search` 支持：

```json
{
  "query": "构建缓存怎么清理",
  "mode": "hybrid",
  "filters": {
    "category_ids": ["engineering"],
    "modules": ["cbb"],
    "docs_versions": ["latest"],
    "entry_types": ["markdown"],
    "keywords": ["cad"]
  },
  "page": 1,
  "page_size": 20
}
```

`mode` 支持：

- `keyword`
- `semantic`
- `hybrid`

Hybrid 排序建议：

```text
final_score = keyword_score * 0.6 + semantic_score * 0.4
```

可配置：

```env
HYBRID_KEYWORD_WEIGHT=0.6
HYBRID_SEMANTIC_WEIGHT=0.4
```

### 13.4 搜索筛选条件

- 分类
- 模块
- 文档版本
- package_version
- Entry 类型
- 文档类型
- keywords
- 状态
- Owner
- 是否 legacy

### 13.5 搜索结果展示

- 标题
- 摘要
- 模块
- 分类
- docs_version
- package_version
- entry_type
- owner
- 更新时间
- 状态
- score
- search_mode

### 13.6 搜索索引字段示例

```json
{
  "doc_id": "DemoModule:latest:guide",
  "module_key": "DemoModule",
  "module_name": "DemoModule",
  "docs_version": "latest",
  "package_version": "1.2.3",
  "channel": "default",
  "category_ids": ["cad", "cad.demo"],
  "entry_key": "guide",
  "entry_type": "markdown",
  "title": "模块落地指导",
  "description": "示例模块落地指导",
  "content": "正文内容",
  "path": "/docs/cad/demo-module/latest/guide",
  "source_file": "docs/integration-guide.md",
  "keywords": ["demo", "cad"],
  "owner_group": "cad-team",
  "status": "active",
  "is_default_version": true,
  "updated_at": "2026-06-09T10:00:00+09:00"
}
```

---

## 14. MCP Server 设计

第一阶段必须实现 MCP Server。

MCP Server 用于让 AI 工具安全读取文档平台内容。

MCP Server 不能直接读 MinIO 或 HTML 文件，必须通过平台 API / Registry / Search Service。

### 14.1 MCP 工具

实现以下工具。

#### list_modules

输入：

```json
{
  "category_id": "cad",
  "keyword": "demo"
}
```

输出模块列表：

```json
[
  {
    "module_key": "DemoModule",
    "name": "DemoModule",
    "description": "示例模块",
    "default_version": "latest",
    "package_version": "1.2.3",
    "keywords": ["demo", "cad"]
  }
]
```

#### list_versions

输入：

```json
{
  "module_key": "DemoModule"
}
```

输出版本列表。

#### search_docs

输入：

```json
{
  "query": "模块如何落地",
  "mode": "hybrid",
  "module_key": "DemoModule",
  "docs_version": "latest",
  "limit": 5
}
```

输出搜索结果，包括：

- doc_id
- title
- snippet
- path
- score
- module_key
- docs_version

#### get_doc_page

输入：

```json
{
  "doc_id": "DemoModule:latest:guide"
}
```

输出：

- 文档正文内容
- 标题
- 来源路径
- 模块信息
- 版本信息

### 14.2 MCP 权限

MVP 可以使用 mock user 或 service token。  
但接口设计要预留真实用户权限过滤。

配置：

```env
MCP_ENABLED=true
MCP_TOKEN=dev-token
```

### 14.3 MCP 与搜索共用能力

MCP 的 `search_docs` 必须调用同一个 Search Service，不能单独实现另一套搜索逻辑。

---

## 15. SSO 设计

MVP 预留 OIDC 配置，但可以先实现 mock login。

后续 OIDC 登录流程：

```text
用户访问平台
  → 未登录跳转 SSO
  → 登录成功回调
  → 后端校验 token
  → 创建 session
  → 同步用户信息和用户组
```

用户信息字段：

- user_id
- username
- display_name
- email
- department
- groups
- roles

---

## 16. PostHog 和阅读统计

MVP 预留 PostHog 初始化代码。

第一阶段埋点事件：

- docs_home_view
- docs_module_click
- docs_module_info_open
- docs_page_view
- docs_search
- docs_search_result_click
- docs_version_switch
- docs_source_click
- docs_mcp_search
- docs_mcp_get_page

登录后预留：

```ts
posthog.identify(user.id, {
  name: user.displayName,
  email: user.email,
  department: user.department,
  groups: user.groups,
})
```

平台自身数据库也记录：

- 文档 PV
- 文档 UV
- 近 7 天阅读量
- 近 30 天阅读量
- 搜索关键词
- 无结果搜索词
- 搜索点击
- 热门文档
- 最近更新文档
- MCP 查询日志

---

## 17. 数据库表设计

请实现以下 MVP 表。

### users

```text
id
username
display_name
email
department
created_at
updated_at
```

### groups

```text
id
group_key
name
source
created_at
updated_at
```

### user_groups

```text
user_id
group_id
```

### docs_category

```text
id
parent_id
key
name
description
icon
sort_order
status
created_at
updated_at
```

### docs_module

```text
id
module_key
name
description
owner_group
repo_type
repo_url
default_version_id
visibility
status
package_name
package_version
channel
edition
keywords
maintainers
created_at
updated_at
```

### docs_module_category

```text
module_id
category_id
is_primary
```

### docs_version

```text
id
module_id
docs_version
display_name
version_type
is_default
status
source_branch
package_version
channel
edition
support_status
created_at
updated_at
```

### docs_entry

```text
id
module_id
version_id
entry_key
title
entry_type
builder
source
storage_uri
nav_uri
index_status
is_primary
sort_order
status
created_at
updated_at
```

### docs_release

```text
id
module_id
version_id
release_id
commit_sha
branch
publisher
pipeline_url
build_system
build_id
artifact_version
package_version
storage_uri
status
published_at
created_at
```

### docs_page

```text
id
module_id
version_id
entry_id
release_id
doc_id
title
description
path
source_file
doc_type
status
owner_group
tags
content_text
updated_at
last_verified_at
created_at
```

### docs_page_view

```text
id
page_id
module_id
version_id
user_id
session_id
duration_seconds
scroll_depth
viewed_at
```

### docs_search_log

```text
id
user_id
query
mode
filters_json
result_count
clicked_doc_id
searched_at
```

### docs_embedding

如果使用 pgvector：

```text
id
page_id
doc_id
chunk_id
module_id
version_id
entry_id
content
embedding vector
metadata_json
created_at
updated_at
```

如果不用 pgvector，先用 JSONB 存 embedding：

```text
embedding_json
```

### docs_mcp_log

```text
id
tool_name
user_id
query
input_json
result_count
created_at
```

---

## 18. API 设计 MVP

### 18.1 Auth

```http
GET /api/auth/me
POST /api/auth/mock-login
POST /api/auth/logout
```

预留：

```http
GET /api/auth/login
GET /api/auth/callback
```

### 18.2 Category

```http
GET /api/categories/tree
POST /api/admin/categories
PUT /api/admin/categories/{id}
DELETE /api/admin/categories/{id}
```

### 18.3 Module

```http
GET /api/modules
GET /api/modules/{module_key}
GET /api/modules/{module_key}/info
POST /api/admin/modules
PUT /api/admin/modules/{module_key}
```

### 18.4 Version

```http
GET /api/modules/{module_key}/versions
POST /api/admin/modules/{module_key}/versions
PUT /api/admin/modules/{module_key}/versions/{docs_version}
```

### 18.5 Entry

```http
GET /api/modules/{module_key}/versions/{docs_version}/entries
POST /api/admin/modules/{module_key}/versions/{docs_version}/entries
PUT /api/admin/entries/{entry_id}
DELETE /api/admin/entries/{entry_id}
```

### 18.6 Document

```http
GET /api/docs/{module_key}
GET /api/docs/{module_key}/{docs_version}
GET /api/docs/{module_key}/{docs_version}/{entry_key}
GET /api/docs/{module_key}/{docs_version}/{entry_key}/nav
GET /api/docs/{module_key}/{docs_version}/{entry_key}/*
GET /api/docs/page/{doc_id}
```

### 18.7 Search

```http
POST /api/search
GET /api/search/facets
POST /api/search/reindex
```

### 18.8 Embedding

```http
POST /api/embeddings/reindex
POST /api/embeddings/embed-text
```

### 18.9 Deploy

```http
POST /api/deploy
GET /api/admin/releases
GET /api/admin/releases/{release_id}
POST /api/admin/releases/{release_id}/rollback
```

### 18.10 Analytics

```http
POST /api/analytics/page-view
POST /api/analytics/read-progress
GET /api/admin/analytics/pages
GET /api/admin/analytics/search
GET /api/admin/analytics/mcp
```

---

## 19. MCP Server API

MCP Server 需要以单独进程或 backend 子命令方式运行。

实现：

```bash
docs-mcp-server
```

配置：

```env
DOCS_API_BASE_URL=http://backend:8080
MCP_TOKEN=dev-token
```

MCP Server 通过 HTTP 调用后端 API。

工具：

- list_modules
- list_versions
- search_docs
- get_doc_page

---

## 20. 前端页面

用户侧：

```text
/
首页

/search
搜索页，支持 keyword / semantic / hybrid 模式切换

/docs/:moduleKey
跳转默认版本

/docs/:moduleKey/:docsVersion
模块版本入口页

/docs/:moduleKey/:docsVersion/:entryKey/*
文档阅读页

/me/recent
最近访问

/me/favorites
我的关注
```

管理侧：

```text
/admin
管理首页

/admin/categories
分类管理

/admin/modules
模块管理

/admin/modules/:moduleKey
模块详情

/admin/releases
发布记录

/admin/analytics
阅读统计

/admin/search-logs
搜索日志

/admin/mcp-logs
MCP 日志
```

---

## 21. docsctl MVP

请在 `tools/docsctl` 实现 Go CLI。

命令：

```bash
docsctl validate
docsctl build
docsctl package
docsctl deploy
```

### 21.1 validate

检查：

- docs.yaml 是否存在。
- entries 是否存在。
- entries 中 key/title/type/source 是否存在。
- source 文件或目录是否存在。
- 如果存在 cbb.toml，检查 [package] 是否可解析。
- VuePress entry 如果有 build/output，检查字段存在。

### 21.2 build

第一阶段支持：

- markdown
- static
- vuepress

#### Markdown builder

- 读取 Markdown 文件。
- 转成基础 HTML。
- 生成 nav.json。
- 生成 documents.jsonl。
- 生成 llms.txt。
- 尝试生成 llms-full.txt。
- 复制 assets。

#### Static builder

- 复制已有 HTML 目录。
- 尝试生成简单 nav.json。
- 尝试抽取 documents.jsonl。
- 生成 llms.txt。
- 如果抽取失败，至少生成 entry-level document 记录，确保搜索可见。

#### VuePress builder

- 读取 entry.build 命令。
- 执行 build 命令。
- 从 entry.output 读取构建后的 HTML。
- 复制到标准 site 目录。
- 尝试从源 Markdown 或构建结果抽取 nav.json 和 documents.jsonl。
- 生成 llms.txt。
- 如果抽取失败，至少生成一个 entry-level document 记录，确保搜索可见。

### 21.3 package

生成：

```text
docs-artifact.zip
  site/
  manifest.json
  metadata.json
  nav.json
  documents.jsonl
  embeddings.jsonl 可选
  llms.txt
  llms-full.txt 可选
  assets/
```

metadata.json 需要合并：

- CI 环境变量
- cbb.toml
- docs.yaml entries

### 21.4 deploy

调用：

```http
POST /api/deploy
```

上传 docs-artifact.zip。

---

## 22. GitLab Pipeline 模板，暂时只预留

MVP 可以先不完整实现 Pipeline 模板，但请预留示例文件：

```yaml
include:
  - project: 'devops/docs-ci-templates'
    ref: main
    file: '/templates/docs-deploy.yml'

variables:
  DOCS_MODULE: "DemoModule"
  DOCS_VERSION: "latest"
  DOCS_BUILDER: "markdown"
  DOCS_SOURCE_DIR: "docs"
```

公共 Pipeline 未来执行：

```text
docsctl validate
docsctl build
docsctl package
docsctl deploy
```

---

## 23. MVP 范围

请先实现：

1. Monorepo 项目结构。
2. Go 后端服务。
3. Next.js 前端。
4. docker-compose，包含 PostgreSQL、MinIO、Meilisearch。
5. pgvector 可选，如果方便则加入。
6. 数据库迁移。
7. Mock 登录。
8. 分类树 API 和页面。
9. 模块 API 和模块卡片。
10. 模块 Info 抽屉。
11. 版本、Entry、Release 的基础管理 API。
12. 文档阅读页占位。
13. 搜索 API 和搜索页，支持 keyword / semantic / hybrid 三种模式。
14. EmbeddingProvider 抽象。
15. MockEmbeddingProvider。
16. HTTPEmbeddingProvider。
17. MCP Server，支持 list_modules / list_versions / search_docs / get_doc_page。
18. MCP 查询日志。
19. PostHog 初始化位置。
20. docsctl 的 validate/build/package 基础能力。
21. cbb.toml 解析能力。
22. markdown builder。
23. static html builder。
24. vuepress builder。
25. 标准文档包中必须包含 llms.txt。
26. README，说明如何本地启动。

暂时不要实现：

- SVN builder-server
- OpenAPI builder
- Doxygen builder
- Javadoc builder
- Fumadocs/Docusaurus builder
- 复杂权限
- 文档质量评分
- 旧路径 redirect

---

## 24. 代码要求

1. 结构清晰。
2. 后端 API 有统一错误处理。
3. 数据库迁移可重复执行。
4. 前端组件拆分合理。
5. docker-compose 一键启动。
6. README 写清楚本地启动步骤。
7. Mock 数据要能展示首页、分类、模块卡片、Info 抽屉、搜索页、MCP 示例。
8. 不要过度设计，但语义搜索、MCP、VuePress builder、llms.txt 的架构必须在第一阶段跑通。

---

## 25. 第一轮交付目标

第一轮交付时，请确保以下内容可以运行：

1. `docker-compose up` 可以启动 PostgreSQL、MinIO、Meilisearch、backend、frontend。
2. 打开前端首页可以看到分类树、模块卡片、Info 抽屉。
3. 可以 mock 登录。
4. 可以访问搜索页，并切换 keyword / semantic / hybrid。
5. 后端有对应 REST API。
6. MCP Server 可以启动，并能调用 list_modules、list_versions、search_docs、get_doc_page。
7. docsctl 可以解析 cbb.toml 和 docs.yaml。
8. docsctl 可以构建 Markdown 示例文档。
9. docsctl 可以生成 docs-artifact.zip。
10. docs-artifact.zip 包含 site、manifest.json、metadata.json、nav.json、documents.jsonl、llms.txt。
