export type RuntimeConfig = {
  apiBaseUrl: string;
  krokiUrl: string;
  gitlabCiTemplateInclude: string;
  docsctlUrl: string;
  posthogKey: string;
  posthogHost: string;
  posthogEnableLocal: boolean;
};

type RuntimeConfigWindow = Partial<
  Omit<RuntimeConfig, "posthogEnableLocal"> & {
    posthogEnableLocal: boolean | string;
  }
>;

declare global {
  interface Window {
    __MODEX_RUNTIME_CONFIG__?: RuntimeConfigWindow;
  }
}

const SERVER_DEFAULT_API_BASE_URL = "http://localhost:8671";
const DEFAULT_KROKI_URL = "https://kroki.io";
const DEFAULT_POSTHOG_HOST = "https://app.posthog.com";
const DEFAULT_DOCSCTL_URL = "https://github.com/modex/modex/releases/latest/download/docsctl-linux-amd64";
const DEFAULT_GITLAB_CI_TEMPLATE_INCLUDE = `include:
  - project: "songkwon/modex-fscut"
    ref: "main"
    file: "deploy/ci/modex-docs.gitlab-ci.yml"`;
const LEGACY_REMOTE_GITLAB_CI_TEMPLATE_URL = "https://raw.githubusercontent.com/songkwon/modex/main/deploy/ci/modex-docs.gitlab-ci.yml";

function envValue(name: string, legacyName: string, fallback = ""): string {
  if (typeof window !== "undefined") return fallback;
  return process.env[name] || process.env[legacyName] || fallback;
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}

function normalizeMultiline(value: string): string {
  return value.replace(/\\n/g, "\n").trim();
}

function defaultApiBaseURL(): string {
  return typeof window === "undefined" ? SERVER_DEFAULT_API_BASE_URL : "";
}

function normalizeGitLabCiTemplateInclude(value: string): string {
  const normalized = normalizeMultiline(value);
  if (normalized.includes(LEGACY_REMOTE_GITLAB_CI_TEMPLATE_URL)) {
    return DEFAULT_GITLAB_CI_TEMPLATE_INCLUDE;
  }
  return normalized;
}

function isLoopbackHost(hostname: string): boolean {
  return hostname === "localhost" || hostname === "127.0.0.1" || hostname === "::1";
}

function normalizeApiBaseURL(value: string): string {
  const trimmed = trimTrailingSlash(value);
  if (typeof window === "undefined" || trimmed === "") return trimmed;
  try {
    const configured = new URL(trimmed);
    if (isLoopbackHost(configured.hostname) && !isLoopbackHost(window.location.hostname)) {
      if (window.location.protocol === "http:" && configured.port && configured.port !== window.location.port) {
        return `${window.location.protocol}//${window.location.hostname}:${configured.port}`;
      }
      return "";
    }
  } catch {
    return trimmed;
  }
  return trimmed;
}

function boolValue(value: boolean | string | undefined): boolean {
  if (typeof value === "boolean") return value;
  return value === "true";
}

function browserConfig(): RuntimeConfigWindow {
  if (typeof window === "undefined") return {};
  return window.__MODEX_RUNTIME_CONFIG__ || {};
}

export function runtimeConfig(): RuntimeConfig {
  const cfg = browserConfig();
  return {
    apiBaseUrl: normalizeApiBaseURL(
      cfg.apiBaseUrl || envValue("MODEX_PUBLIC_API_BASE_URL", "NEXT_PUBLIC_API_BASE_URL", defaultApiBaseURL())
    ),
    krokiUrl: trimTrailingSlash(
      cfg.krokiUrl || envValue("MODEX_PUBLIC_KROKI_URL", "NEXT_PUBLIC_KROKI_URL", DEFAULT_KROKI_URL)
    ),
    gitlabCiTemplateInclude: normalizeGitLabCiTemplateInclude(
      cfg.gitlabCiTemplateInclude ||
        envValue("MODEX_PUBLIC_GITLAB_CI_TEMPLATE_INCLUDE", "NEXT_PUBLIC_GITLAB_CI_TEMPLATE_INCLUDE", DEFAULT_GITLAB_CI_TEMPLATE_INCLUDE)
    ),
    docsctlUrl: trimTrailingSlash(
      cfg.docsctlUrl || envValue("MODEX_PUBLIC_DOCSCTL_URL", "NEXT_PUBLIC_DOCSCTL_URL", DEFAULT_DOCSCTL_URL)
    ),
    posthogKey: cfg.posthogKey || envValue("MODEX_PUBLIC_POSTHOG_KEY", "NEXT_PUBLIC_POSTHOG_KEY"),
    posthogHost: trimTrailingSlash(
      cfg.posthogHost || envValue("MODEX_PUBLIC_POSTHOG_HOST", "NEXT_PUBLIC_POSTHOG_HOST", DEFAULT_POSTHOG_HOST)
    ),
    posthogEnableLocal: boolValue(
      cfg.posthogEnableLocal ?? envValue("MODEX_PUBLIC_POSTHOG_ENABLE_LOCAL", "NEXT_PUBLIC_POSTHOG_ENABLE_LOCAL", "false")
    )
  };
}

export function publicApiBaseURL(): string {
  return runtimeConfig().apiBaseUrl;
}

export function publicKrokiURL(): string {
  return runtimeConfig().krokiUrl;
}

export function gitlabCiTemplateInclude(): string {
  return runtimeConfig().gitlabCiTemplateInclude;
}

export function publicDocsctlURL(): string {
  return runtimeConfig().docsctlUrl;
}
