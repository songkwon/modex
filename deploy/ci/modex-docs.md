# Modex GitLab CI 文档同步

在文档仓库的 `.gitlab-ci.yml` 中 include 模板，并只填写必要变量：

```yaml
include:
  - project: "songkwon/modex-fscut"
    ref: "main"
    file: "deploy/ci/modex-docs.gitlab-ci.yml"

variables:
  MODEX_DEPLOY_URL: "https://modex.example.com/api/deploy"
```

在 GitLab `Settings > CI/CD > Variables` 中添加：

```text
MODEX_DEPLOY_TOKEN=<从 Modex 文档源复制的 Deploy Token>
```

建议设置为 Masked，生产分支可同时设置为 Protected。不要把 token 写进仓库。

## 默认行为

模板会在默认分支自动执行 `docsctl deploy`，内部包含 validate、build、package、upload。

`docsctl` 会自动识别常见文档项目。需要手动指定时，`DOCS_BUILDER` 支持：

| DOCS_BUILDER | 说明 |
| --- | --- |
| `vitepress` | VitePress |
| `vuepress` | VuePress |
| `fumadocs` | Fumadocs |
| `docusaurus` | Docusaurus |
| `mkdocs` | MkDocs |
| `honkit` | HonKit |
| `gitbook` | GitBook |
| `markdown` | 普通 Markdown 文档 |
| `static` | 已构建好的 HTML 静态目录 |

大多数仓库不需要配置 `DOCS_BUILDER`、`DOCS_BUILD` 或 `DOCS_OUTPUT`。

以下情况建议手动指定 `DOCS_BUILDER`：

- 仓库里同时存在多种文档框架配置，例如既有 VitePress 又有 MkDocs。
- 文档不在仓库根目录，且 `DOCS_SOURCE_DIR` 指向的子目录无法被自动识别。
- 已构建好的静态目录需要直接同步，例如 `dist/` 或 `public/`，这时用 `DOCS_BUILDER: "static"`。
- 框架配置文件使用了非默认位置或特殊命名，自动识别不到。
- 自动识别结果和预期不一致，CI 日志里的 builder 类型不对。

## 可选变量

手动指定 builder 时，通常一起配置构建命令和产物目录：

```yaml
variables:
  DOCS_SOURCE_DIR: "docs"
  DOCS_BUILDER: "vitepress"
  DOCS_BUILD: "npm ci && npm run docs:build"
  DOCS_OUTPUT: "docs/.vitepress/dist"
```

已构建好的静态目录不需要 `DOCS_BUILD` / `DOCS_OUTPUT`：

```yaml
variables:
  MODEX_DEPLOY_URL: "https://modex.example.com/api/deploy"
  DOCS_BUILDER: "static"
  DOCS_SOURCE_DIR: "dist"
```

## 排除非文档 Markdown

如果仓库里有 `CLAUDE.md`、`AGENTS.md`、草稿、内部备注等不希望同步到 Modex 的 Markdown，可以在 `DOCS_SOURCE_DIR` 所在目录放 `.modexignore`：

`docsctl` 默认会过滤常见非文档 Markdown 文件：

```text
CLAUDE.md
AGENTS.md
GEMINI.md
QWEN.md
CURSOR.md
COPILOT.md
```

也会默认跳过 `.git/`、`.modex/`、`.cursor/`、`.claude/`、`.github/`、`.vscode/`、`node_modules/`、`dist/`、`build/` 等工具或构建目录。

如果还有仓库自定义的草稿、临时文档或内部备注，再用 `.modexignore` 补充：

```gitignore
CLAUDE.md
AGENTS.md
drafts/
*.local.md
**/*.draft.md
```

规则按同步根目录匹配；如果 `DOCS_SOURCE_DIR` 是仓库根目录，排除 `docs/runtime/*.mdx` 需要写完整相对路径；如果 `DOCS_SOURCE_DIR` 是 `docs`，则写 `runtime/*.mdx` 即可。

## split 挂载方式

后台文档源的挂载方式为 `split` 时，GitLab CI 仍然使用同一套模板和同一个 deploy token，不需要额外配置 CI 变量。`docsctl` 上传标准 artifact 后，Modex 后端会根据 token 找到文档源配置，并在入库时把 Markdown 按顶层目录拆成多个入口。

`split` 只对普通 Markdown 文档源生效；VitePress、VuePress、Fumadocs、Docusaurus、MkDocs、静态站点等编译型文档固定按 `single` 处理。

其他低频覆盖项：

```yaml
variables:
  DOCS_VERSION: "latest"
  MODEX_DOCS_IMAGE: "python:3.12-bookworm"
  MODEX_DOCSCTL_URL: "https://your-registry.example.com/docsctl-linux-amd64"
```

## 多目录仓库

同一个仓库发布多个文档源时，可以建多个 job，并为每个 job 配置对应文档源的 `MODEX_DEPLOY_TOKEN` 和 `DOCS_SOURCE_DIR`。

```yaml
rd-standards:
  extends: .modex-docs-base
  variables:
    DOCS_SOURCE_DIR: "docs/standards"
    MODEX_DEPLOY_TOKEN: "$MODEX_DEPLOY_TOKEN_RD_STANDARDS"

rd-guides:
  extends: .modex-docs-base
  variables:
    DOCS_SOURCE_DIR: "docs/guides"
    MODEX_DEPLOY_TOKEN: "$MODEX_DEPLOY_TOKEN_RD_GUIDES"
```
