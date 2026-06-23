export type RuntimeConfig = {
  apiBaseUrl: string;
  krokiUrl: string;
  gitlabCiTemplateInclude: string;
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

const DEFAULT_API_BASE_URL = "http://localhost:8671";
const DEFAULT_KROKI_URL = "https://kroki.io";
const DEFAULT_POSTHOG_HOST = "https://app.posthog.com";
const DEFAULT_GITLAB_CI_TEMPLATE_INCLUDE = `include:
  - project: "songkwon/modex-fscut"
    ref: "main"
    file: "deploy/ci/modex-docs.gitlab-ci.yml"`;

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
    apiBaseUrl: trimTrailingSlash(
      cfg.apiBaseUrl || envValue("MODEX_PUBLIC_API_BASE_URL", "NEXT_PUBLIC_API_BASE_URL", DEFAULT_API_BASE_URL)
    ),
    krokiUrl: trimTrailingSlash(
      cfg.krokiUrl || envValue("MODEX_PUBLIC_KROKI_URL", "NEXT_PUBLIC_KROKI_URL", DEFAULT_KROKI_URL)
    ),
    gitlabCiTemplateInclude: normalizeMultiline(
      cfg.gitlabCiTemplateInclude ||
        envValue("MODEX_PUBLIC_GITLAB_CI_TEMPLATE_INCLUDE", "NEXT_PUBLIC_GITLAB_CI_TEMPLATE_INCLUDE", DEFAULT_GITLAB_CI_TEMPLATE_INCLUDE)
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
