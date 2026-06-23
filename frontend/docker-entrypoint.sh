#!/bin/sh
set -eu

runtime_config_file="${MODEX_RUNTIME_CONFIG_FILE:-/app/public/runtime-env.js}"
export MODEX_RUNTIME_CONFIG_FILE="$runtime_config_file"

node <<'NODE'
const fs = require("fs");
const path = require("path");

function pick(primary, legacy, fallback = "") {
  return process.env[primary] ?? process.env[legacy] ?? fallback;
}

const config = {
  apiBaseUrl: pick("MODEX_PUBLIC_API_BASE_URL", "NEXT_PUBLIC_API_BASE_URL", "http://localhost:8671").replace(/\/+$/, ""),
  krokiUrl: pick("MODEX_PUBLIC_KROKI_URL", "NEXT_PUBLIC_KROKI_URL", "https://kroki.io").replace(/\/+$/, ""),
  gitlabCiTemplateInclude: pick(
    "MODEX_PUBLIC_GITLAB_CI_TEMPLATE_INCLUDE",
    "NEXT_PUBLIC_GITLAB_CI_TEMPLATE_INCLUDE",
    'include:\\n  - project: "songkwon/modex-fscut"\\n    ref: "main"\\n    file: "deploy/ci/modex-docs.gitlab-ci.yml"'
  ).replace(/\\n/g, "\n").trim(),
  posthogKey: pick("MODEX_PUBLIC_POSTHOG_KEY", "NEXT_PUBLIC_POSTHOG_KEY"),
  posthogHost: pick("MODEX_PUBLIC_POSTHOG_HOST", "NEXT_PUBLIC_POSTHOG_HOST", "https://app.posthog.com").replace(/\/+$/, ""),
  posthogEnableLocal: pick("MODEX_PUBLIC_POSTHOG_ENABLE_LOCAL", "NEXT_PUBLIC_POSTHOG_ENABLE_LOCAL", "false") === "true",
};

const target = process.env.MODEX_RUNTIME_CONFIG_FILE;
fs.mkdirSync(path.dirname(target), { recursive: true });
fs.writeFileSync(target, `window.__MODEX_RUNTIME_CONFIG__ = ${JSON.stringify(config, null, 2)};\n`);
NODE

exec "$@"
