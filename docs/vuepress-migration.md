# VuePress 文档迁移方案

## 目标

现有 RD 和各文档集散地已经是独立 VuePress 项目时，不需要重写文档内容。迁移目标是让每个项目保留原来的目录、主题和构建命令，只补充 Modex 发布协议。

## 推荐路径

1. 在 VuePress 项目根目录运行：

   ```bash
   docsctl init
   ```

2. 检查生成的 `docs.yaml`：

   ```yaml
   entries:
     - key: guide
       title: VuePress 文档
       type: vuepress
       source: docs
       build: npm run docs:build
       output: docs/.vuepress/dist
   ```

3. 在 CI 中执行：

   ```bash
   docsctl validate
   docsctl build
   docsctl package
   docsctl deploy
   ```

4. 在 Modex Registry 中维护分类、Owner、权限、默认版本和展示信息。

## RD 聚合站迁移

RD 本身如果也是 VuePress 项目，可以作为一个普通模块发布：

```env
DOCS_MODULE=RD
DOCS_VERSION=latest
DOCS_SOURCE_DIR=.
```

它仍然可以保留原来的导航结构。Modex 只接收构建后的标准文档包，并将内容纳入搜索、统计和 MCP 读取。

## 多文档集散地迁移

每个独立 VuePress 项目独立接入：

- 每个项目各自维护 `docs.yaml`
- 每个项目通过公共 Pipeline 发布
- 平台 Registry 负责统一分类和权限
- 不要求所有项目使用同一个 VuePress 主题

可以先在旧 Wiki 或 GitLab group checkout 根目录做本地扫描：

```bash
DOCS_SOURCE_DIR=/path/to/wiki-root docsctl discover
```

确认列表后批量写入 `docs.yaml`：

```bash
DOCS_SOURCE_DIR=/path/to/wiki-root DOCS_DISCOVER_WRITE=true docsctl discover
```

如果要覆盖已有配置：

```bash
DOCS_SOURCE_DIR=/path/to/wiki-root DOCS_DISCOVER_WRITE=true DOCS_INIT_FORCE=true docsctl discover
```

## 兼容规则

- `docs.yaml` 只声明入口、构建命令和输出目录。
- 模块名、Owner、分类、权限不放在 `docs.yaml` 中。
- 如果项目已有 `cbb.toml`，docsctl 会读取 package metadata。
- 如果没有 `cbb.toml`，CI 可以用 `DOCS_MODULE`、`DOCS_VERSION` 等变量覆盖。

## 后续自动化

本地批量扫描已经由 `docsctl discover` 支持。进一步可以接入 GitLab 自动化：

```text
扫描 GitLab group
  -> 找 package.json + .vuepress/config.*
  -> 调用 docsctl discover 生成 docs.yaml
  -> 自动 MR 提交 docs.yaml
  -> 接入公共 docs pipeline
```
