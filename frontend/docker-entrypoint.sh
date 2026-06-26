#!/bin/sh
set -eu

runtime_config_file="${MODEX_RUNTIME_CONFIG_FILE:-/app/public/runtime-env.js}"
export MODEX_RUNTIME_CONFIG_FILE="$runtime_config_file"

detect_brand_file() {
  name="$1"
  for ext in svg png webp jpg jpeg ico; do
    if [ -f "/app/public/brand/${name}.${ext}" ]; then
      printf "/brand/%s.%s" "$name" "$ext"
      return 0
    fi
  done
  return 1
}

if [ -z "${MODEX_PUBLIC_LOGO_URL:-}" ] && [ -z "${NEXT_PUBLIC_LOGO_URL:-}" ]; then
  if detected="$(detect_brand_file logo)"; then
    export MODEX_PUBLIC_LOGO_URL="$detected"
  fi
fi

if [ -z "${MODEX_PUBLIC_LOGO_LIGHT_URL:-}" ] && [ -z "${NEXT_PUBLIC_LOGO_LIGHT_URL:-}" ]; then
  if detected="$(detect_brand_file logo-light)"; then
    export MODEX_PUBLIC_LOGO_LIGHT_URL="$detected"
  elif [ -n "${MODEX_PUBLIC_LOGO_URL:-}" ]; then
    export MODEX_PUBLIC_LOGO_LIGHT_URL="$MODEX_PUBLIC_LOGO_URL"
  fi
fi

if [ -z "${MODEX_PUBLIC_LOGO_DARK_URL:-}" ] && [ -z "${NEXT_PUBLIC_LOGO_DARK_URL:-}" ]; then
  if detected="$(detect_brand_file logo-dark)"; then
    export MODEX_PUBLIC_LOGO_DARK_URL="$detected"
  elif [ -n "${MODEX_PUBLIC_LOGO_URL:-}" ]; then
    export MODEX_PUBLIC_LOGO_DARK_URL="$MODEX_PUBLIC_LOGO_URL"
  fi
fi

if [ -z "${MODEX_PUBLIC_FAVICON_URL:-}" ] && [ -z "${NEXT_PUBLIC_FAVICON_URL:-}" ]; then
  if detected="$(detect_brand_file favicon)"; then
    export MODEX_PUBLIC_FAVICON_URL="$detected"
  fi
fi

node <<'NODE'
const fs = require("fs");
const path = require("path");

function pick(primary, legacy, fallback = "") {
  return process.env[primary] ?? process.env[legacy] ?? fallback;
}

const defaultGitlabCiTemplateInclude = 'include:\\n  - project: "songkwon/modex-fscut"\\n    ref: "main"\\n    file: "deploy/ci/modex-docs.gitlab-ci.yml"';
const legacyRemoteGitlabCiTemplateUrl = "https://raw.githubusercontent.com/songkwon/modex/main/deploy/ci/modex-docs.gitlab-ci.yml";

function normalizeGitlabCiTemplateInclude(value) {
  const normalized = value.replace(/\\n/g, "\n").trim();
  if (normalized.includes(legacyRemoteGitlabCiTemplateUrl)) {
    return defaultGitlabCiTemplateInclude.replace(/\\n/g, "\n").trim();
  }
  return normalized;
}

const config = {
  appTitle: pick("MODEX_PUBLIC_APP_TITLE", "NEXT_PUBLIC_APP_TITLE", "Modex"),
  logoUrl: pick("MODEX_PUBLIC_LOGO_URL", "NEXT_PUBLIC_LOGO_URL", "/logo.svg"),
  logoLightUrl: pick("MODEX_PUBLIC_LOGO_LIGHT_URL", "NEXT_PUBLIC_LOGO_LIGHT_URL", pick("MODEX_PUBLIC_LOGO_URL", "NEXT_PUBLIC_LOGO_URL", "/logo.svg")),
  logoDarkUrl: pick("MODEX_PUBLIC_LOGO_DARK_URL", "NEXT_PUBLIC_LOGO_DARK_URL", pick("MODEX_PUBLIC_LOGO_URL", "NEXT_PUBLIC_LOGO_URL", "/logo.svg")),
  faviconUrl: pick("MODEX_PUBLIC_FAVICON_URL", "NEXT_PUBLIC_FAVICON_URL", "/icon.svg"),
  apiBaseUrl: pick("MODEX_PUBLIC_API_BASE_URL", "NEXT_PUBLIC_API_BASE_URL", "http://localhost:8671").replace(/\/+$/, ""),
  krokiUrl: pick("MODEX_PUBLIC_KROKI_URL", "NEXT_PUBLIC_KROKI_URL", "https://kroki.io").replace(/\/+$/, ""),
  gitlabCiTemplateInclude: normalizeGitlabCiTemplateInclude(pick(
    "MODEX_PUBLIC_GITLAB_CI_TEMPLATE_INCLUDE",
    "NEXT_PUBLIC_GITLAB_CI_TEMPLATE_INCLUDE",
    defaultGitlabCiTemplateInclude
  )),
  docsctlUrl: pick(
    "MODEX_PUBLIC_DOCSCTL_URL",
    "NEXT_PUBLIC_DOCSCTL_URL",
    "https://github.com/modex/modex/releases/latest/download/docsctl-linux-amd64"
  ).replace(/\/+$/, ""),
  posthogKey: pick("MODEX_PUBLIC_POSTHOG_KEY", "NEXT_PUBLIC_POSTHOG_KEY"),
  posthogHost: pick("MODEX_PUBLIC_POSTHOG_HOST", "NEXT_PUBLIC_POSTHOG_HOST", "https://app.posthog.com").replace(/\/+$/, ""),
  posthogEnableLocal: pick("MODEX_PUBLIC_POSTHOG_ENABLE_LOCAL", "NEXT_PUBLIC_POSTHOG_ENABLE_LOCAL", "false") === "true",
};

const target = process.env.MODEX_RUNTIME_CONFIG_FILE;
fs.mkdirSync(path.dirname(target), { recursive: true });
fs.writeFileSync(target, `window.__MODEX_RUNTIME_CONFIG__ = ${JSON.stringify(config, null, 2)};\n`);
NODE

exec "$@"
