// Analytics helper. PostHog integration is wired here behind env config so the
// rest of the app can capture events without importing posthog directly.
//
// To enable PostHog, set MODEX_PUBLIC_POSTHOG_KEY at container runtime and
// optionally MODEX_PUBLIC_POSTHOG_HOST.

import { runtimeConfig } from "@/lib/runtime-config";

export type AnalyticsEvent =
  | "docs_home_view"
  | "docs_module_click"
  | "docs_module_info_open"
  | "docs_page_view"
  | "docs_page_read"
  | "docs_search"
  | "docs_search_result_click"
  | "docs_version_switch"
  | "docs_feedback"
  | "docs_source_click"
  | "docs_mcp_search"
  | "docs_mcp_get_page";

let initialized = false;

function analyticsEnabled(): boolean {
  const cfg = runtimeConfig();
  if (typeof window === "undefined" || !cfg.posthogKey) return false;
  const local = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1" || window.location.hostname === "::1";
  return !local || cfg.posthogEnableLocal;
}

export function initAnalytics(): void {
  if (initialized || !analyticsEnabled()) return;
  const cfg = runtimeConfig();
  const key = cfg.posthogKey;
  initialized = true;
  import("posthog-js").then((posthog) => {
    posthog.default.init(key, { api_host: cfg.posthogHost });
  }).catch(() => {
    initialized = false;
  });
}

export function identify(user: { id: string; displayName?: string; email?: string; department?: string }): void {
  if (!analyticsEnabled()) return;
  import("posthog-js").then((posthog) => {
    posthog.default.identify(user.id, {
      name: user.displayName,
      email: user.email,
      department: user.department,
    });
  }).catch(() => {});
}

export function capture(event: AnalyticsEvent, props: Record<string, unknown> = {}): void {
  if (typeof window === "undefined") return;
  if (analyticsEnabled()) {
    import("posthog-js").then((posthog) => {
      posthog.default.capture(event, props);
    }).catch(() => {});
  }
  if (process.env.NODE_ENV !== "production") {
    // Helpful during MVP development before PostHog is enabled.
    console.debug("[analytics]", event, props);
  }
}

// sessionId returns a per-browser-session identifier for feedback records.
export function sessionId(): string {
  if (typeof window === "undefined") return "ssr";
  const KEY = "modex_session_id";
  let id = window.sessionStorage.getItem(KEY);
  if (!id) {
    id = `s-${Math.random().toString(36).slice(2)}-${Date.now().toString(36)}`;
    window.sessionStorage.setItem(KEY, id);
  }
  return id;
}
