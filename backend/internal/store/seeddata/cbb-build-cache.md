---
title: 构建缓存清理
description: CBB 构建缓存清理和常见构建问题排查。
---

当依赖缓存、编译缓存或 CI 工作区残留导致构建异常时，可按本页步骤清理。

<Warning>清理缓存会导致下一次构建变慢，请在确认存在缓存污染时再执行。</Warning>

## 清理步骤

<Steps>
  <Step title="清理本地缓存">
    ```bash
    cbb cache clean --all
    ```
  </Step>
  <Step title="重新拉取依赖">
    ```bash
    cbb deps sync --force
    ```
  </Step>
  <Step title="重新构建">
    ```bash
    cbb build --no-cache
    ```
  </Step>
</Steps>

## 不同环境

<Tabs>
  <Tab title="本地">删除 `.cbb/cache` 目录后重新构建。</Tab>
  <Tab title="CI">在流水线中清理 runner 工作区缓存卷。</Tab>
</Tabs>

<Check>构建成功后产物哈希应与上一个稳定版本一致（除非源码变更）。</Check>
