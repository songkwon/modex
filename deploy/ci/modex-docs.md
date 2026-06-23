# Modex GitLab CI 文档同步

在文档仓库的 `.gitlab-ci.yml` 中 include 模板，并只填写必要变量：

```yaml
include:
  - remote: "https://raw.githubusercontent.com/songkwon/modex/main/deploy/ci/modex-docs.gitlab-ci.yml"

variables:
  MODEX_MODULE_KEY: "rd-doc"
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
  MODEX_MODULE_KEY: "rd-doc"
  MODEX_DEPLOY_URL: "https://modex.example.com/api/deploy"
  DOCS_BUILDER: "static"
  DOCS_SOURCE_DIR: "dist"
```

其他低频覆盖项：

```yaml
variables:
  DOCS_VERSION: "latest"
  MODEX_DOCS_IMAGE: "python:3.12-bookworm"
  MODEX_DOCSCTL_URL: "https://your-registry.example.com/docsctl-linux-amd64"
```

## 多目录仓库

同一个仓库发布多个文档源时，可以建多个 job，并分别设置 `MODEX_MODULE_KEY` 和 `DOCS_SOURCE_DIR`。

```yaml
rd-standards:
  extends: .modex-docs-base
  variables:
    MODEX_MODULE_KEY: "rd-standards"
    DOCS_SOURCE_DIR: "docs/standards"

rd-guides:
  extends: .modex-docs-base
  variables:
    MODEX_MODULE_KEY: "rd-guides"
    DOCS_SOURCE_DIR: "docs/guides"
```
