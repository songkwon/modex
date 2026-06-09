# 模块维护说明

## 总体架构

模块文档随代码仓库维护，发布后由 Modex Registry 统一治理。

## 核心流程

docsctl validate、build、package、deploy 依次完成校验、构建、打包和发布。

## 质量与可维护性

标准文档包必须包含 site、manifest.json、metadata.json、nav.json、documents.jsonl 和 llms.txt。
