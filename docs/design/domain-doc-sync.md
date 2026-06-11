# 领域 × 文档仓库同步方案

> 状态：设计已确认，待实现
> 关联：`tools/docsctl`、`backend/internal/store`(Category/Module)、`backend/internal/api/server.go`(deploy/webhook)、`docs/pipeline/docs-deploy.example.yml`

把外部文档/代码仓库（VitePress、VuePress、Fumadocs、纯 Markdown…）接入 modex，挂到能力域树的某个节点下，点击领域即可阅读，并自动进入搜索/AI 索引。设计目标：**modex 本身开源、通用，不依赖文档仓库自带 modex 配置文件。**

---

## 1. 概念模型（两层）

**第 1 层 · 领域树（平台 IA）** —— 完全在 modex admin 维护，任意深度，与任何仓库无关。对应 `store.Category`（已是 `ParentID` + 递归 `Children` 的树，dotted-key 约定如 `standards.tools.version`）。

**第 2 层 · 文档源（一个仓库）** —— 在 admin 登记：`RepoURL + Branch + Type + DeployToken + 一个锚点领域节点`。对应 `store.Module`（已有 `RepoURL/GitLabBranch/GitLabPath/DeployToken/CategoryIDs/CategoryPath`，形状基本就位）。

```
领域树 (Category, 平台维护)
└─ 研发规范 (standards)
   ├─ 基础规范 (standards.standard)   ← 锚点：绑定 repo-A
   └─ 工具规范 (standards.tools)
      └─ 版本管理 (standards.tools.version) ← 锚点：绑定 repo-B
应用 (app)                              ← 锚点：绑定 repo-C
```

---

## 2. 绑定规则（已确认）

1. **一个仓库 → 一个锚点节点。** 锚点可以是顶层领域、子领域或更深层级，深度任意可配（admin 里"领域下创建子领域"已支持）。
2. **不允许一个仓库散射到多个顶层能力域。** 也不做"任意子树 → 任意域"的子页面级散射绑定（太碎、不可解释）。
3. **`mount` 开关**控制锚点下如何展开仓库内容：
   - `single`：整个仓库是锚点下的一篇文档，仓库自身导航作为文档内部目录。
   - `split`：仓库的顶层文件夹各自变成锚点下自动创建（upsert）的子域。
4. **`mount` 与渲染方式的交互（关键约束）：**
   - **编译型仓库固定 `single`** —— 框架 build 出来是一个整体站点（自带路由），无法切成多个子域。
   - **`split` 只对 `markdown` 型有意义** —— 此时渲染权在 modex，才能把文件夹拆成子域。

> rd-doc 是 VitePress → 只能 `single`，绑到某个领域锚点，点进去靠它自己的 sidebar 走内部（standard/tools/sparklers 是站内页面，不是 modex 子域）。想让它们成为独立子域，需把 rd-doc 拆成多个仓库分别绑定。

---

## 3. 文档源类型与渲染

每个文档源有一个 `type`，决定同步时的处理：

| type | 同步动作 | 展示渲染 | 内部导航来源 |
|---|---|---|---|
| `vitepress` / `vuepress` / `fumadocs` | 跑框架 `build` | 静态 HTML 托管 MinIO，点领域 = 进站点 | **框架自带**（手写分组/排序原样保留）|
| `markdown` | 不编译 | modex 自带阅读器渲染 | modex 按文件夹 + frontmatter（`title`/`order`）自动生成 |

**两条路都额外吐一份纯文本索引**（`documents.jsonl` + `llms.txt`）给搜索与 AI——一次同步两个产物：给人看的（HTML / modex 渲染）+ 给检索的（纯文本）。这二者解耦，docsctl 现已具备。

---

## 4. 同步方向：默认 B（仓库 CI 推），A（modex 拉取）可选

| | A. modex 拉取编译 | **B. 仓库 CI 编译再推（默认）** |
|---|---|---|
| 触发 | webhook/定时 → modex clone → 跑框架 build | 仓库 push → 仓库 CI 跑 docsctl → `POST /api/deploy` |
| modex 负担 | 需装齐各框架工具链 + 沙箱跑别人构建脚本（RCE 面）、镜像重 | **零工具链、不执行外部代码** |
| 仓库负担 | 零 | 加一个 `include` 的 CI job |

选 B 的原因：开源用户零额外运维即可用；modex 不背多框架工具链与沙箱。A 留作"想要 webhook 全自动"的可选 worker（后续再做）。

### B 模式时序

```
push → GitLab CI(include 模板)
        └─ docsctl build   (按 DOCS_BUILDER 跑框架 build 或直接收集 markdown)
        └─ docsctl package (打 zip：site/ + documents.jsonl + llms.txt + manifest)
        └─ docsctl deploy  (POST /api/deploy, 带 X-Modex-Deploy-Token)
modex /api/deploy
        └─ ParseZip → SiteHTML/SiteFiles 进 MinIO；records 进搜索/向量索引
        └─ 按 Module 绑定的锚点 CategoryIDs 归域
前端：点领域锚点 → 若有托管站点则深链进 MinIO 静态站；markdown 型则进 modex 阅读器
```

---

## 5. GitLab CI 模板（B 模式的核心交付）

目标：仓库 `include` 一个模板 + 设几个变量即可，**不需要在仓库里维护 `docs.yaml`**（docsctl 从环境变量合成单 entry 配置）。

`deploy/ci/modex-docs.gitlab-ci.yml`（modex 仓库提供，开源用户镜像/引用）：

```yaml
# 仓库侧 .gitlab-ci.yml
include:
  - remote: 'https://raw.githubusercontent.com/<org>/modex/main/deploy/ci/modex-docs.gitlab-ci.yml'

variables:
  MODEX_MODULE_KEY: "rd-doc"              # 对应 modex 后台登记的文档源 key（锚点在后台配，CI 不管）
  DOCS_BUILDER:     "vitepress"           # vitepress | vuepress | fumadocs | markdown
  DOCS_SOURCE_DIR:  "docs"
  DOCS_BUILD:       "npm ci && npm run docs:build"   # markdown 型留空
  DOCS_OUTPUT:      "docs/.vitepress/dist"            # markdown 型留空
  MODEX_DEPLOY_URL: "https://modex.example.com/api/deploy"
  # MODEX_DEPLOY_TOKEN: 在 GitLab CI 变量里设 (Masked + Protected)，勿写进仓库
```

模板内部（modex 维护）大致：

```yaml
modex-docs-deploy:
  image: ghcr.io/<org>/docsctl:latest    # 预装 docsctl + node，免去用户装工具链
  rules:
    - if: '$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH'
  script:
    - docsctl build
    - docsctl package
    - docsctl deploy
  variables:
    DOCS_DEPLOY_URL: "$MODEX_DEPLOY_URL"
    DOCS_DEPLOY_TOKEN: "$MODEX_DEPLOY_TOKEN"
    DOCS_MODULE: "$MODEX_MODULE_KEY"
```

> 锚点领域绑定在 **modex 后台**完成（文档源 → 选锚点节点 + mount），CI 只携带 `MODULE_KEY` 推产物，二者解耦。

---

## 6. 需要的代码改动（增量，模型基本不动）

### docsctl
- **新增 `vitepress` builder**：复用现有 `vuepress`/`fumadocs` 的"跑 `Build` 命令 + 拷 `Output`"路径（[build.go:88 buildCommandEntry](../../tools/docsctl/internal/docs/build.go)）。
- **env 合成配置**：无 `docs.yaml` 时，从 `DOCS_BUILDER/DOCS_SOURCE_DIR/DOCS_BUILD/DOCS_OUTPUT` 合成单 entry，免仓库维护配置文件。
- **修 `deploy()` 缺少 token 头**：当前 [main.go:63](../../tools/docsctl/cmd/docsctl/main.go) 未发送 `X-Modex-Deploy-Token`，B 模式会 403，必补。
- `markdown` 型：递归收集 `.md`，按文件夹 + frontmatter 生成 nav（已有 `extractMDFilesSummary` 雏形，需扩展为带层级的 nav）。

### backend
- **`/api/deploy`**：已支持 per-module DeployToken 校验与 SiteFiles→MinIO（[server.go:524](../../backend/internal/api/server.go)）。需确保入库时用 Module 锚点的 `CategoryIDs` 归域。
- **Module 增字段**：`Type`(vitepress/…/markdown)、`Mount`(single/split)、`AnchorCategoryID`（可直接复用 `CategoryIDs[0]` 作锚点）。
- **`split` 落地**：markdown 型同步时，按仓库顶层文件夹 upsert 子 Category（锚点的子域），各文件夹内容归对应子域。
- **`/api/webhooks/gitlab`**（已存在）：B 模式下仅用于记录/触发审计；真正的 build 在仓库 CI。A 模式才用它触发 modex 拉取。

### admin 前端
- 文档源登记表单：RepoURL / Branch / Type / 生成 DeployToken / **锚点领域选择器（树形，任意深度）** / mount 开关（编译型禁用为 single）。

---

## 7. 层级深度策略

- **数据层不设硬上限**：`Category` 与 `NavItem` 均已递归。
- **UX 建议**：领域树 ≤ 4 层、文档内 nav 分组 ≤ 3 层（与现有 VitePress `sidebarDepth: 5` 一致的"引擎无限、推荐浅"思路，对齐 Mintlify/GitBook）。

---

## 8. rd-doc 落地示例

- rd-doc 是 VitePress、内容异构（规范+工具+sparklers app 模块）。按规则：**整仓 `single`，绑到一个锚点**（如 `研发规范`）。
- CI：`DOCS_BUILDER=vitepress`、`DOCS_BUILD=npm ci && npm run docs:build`、`DOCS_OUTPUT=docs/.vitepress/dist`、`MODEX_MODULE_KEY=rd-doc`。
- 点"研发规范"领域 → 进 rd-doc 的 VitePress 站，standard/tools/sparklers 走站内 sidebar。
- 若想让 version-control / workflow / sparklers 成为 modex 独立子域 → 需把 rd-doc 拆成多仓库分别绑各子域（内容组织问题，非产品限制）。

---

## 9. 分期

1. **P0**：docsctl 修 deploy token 头 + env 合成配置 + vitepress builder；CI 模板 `deploy/ci/modex-docs.gitlab-ci.yml`；deploy 入库按锚点归域。→ rd-doc 可端到端 `single` 接入。
2. **P1**：Module 增 `Type/Mount`；admin 文档源表单 + 锚点选择器。
3. **P2**：markdown 型 modex 自渲染 + 文件夹 nav 自动生成 + `split` 子域 upsert。
4. **P3（可选）**：A 模式 webhook → modex 拉取编译（沙箱 worker）。
</content>
