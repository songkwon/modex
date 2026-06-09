# VuePress 文档站接入

## 概览

这个示例模拟一个 VuePress 文档站。真实项目可以把 `build` 改成 `pnpm docs:build`。

## 发布流程

1. 在仓库维护 `docs.yaml`。
2. 执行 VuePress 构建。
3. docsctl 复制 `docs/.vuepress/dist` 到标准文档包。
