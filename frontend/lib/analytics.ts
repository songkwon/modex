// Analytics helper. PostHog integration is wired here behind env config so the
// rest of the app can capture events without importing posthog directly.
//
// To enable PostHog, set MODEX_PUBLIC_POSTHOG_KEY at container runtime and
// optionally MODEX_PUBLIC_POSTHOG_HOST.

import { runtimeConfig } from "@/lib/runtime-config";
import type { PostHog, PostHogConfig } from "posthog-js";

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
let posthogPromise: Promise<PostHog> | null = null;

const posthogNoRemoteConfig: Pick<
  PostHogConfig,
  | "advanced_disable_decide"
  | "advanced_disable_flags"
  | "advanced_disable_feature_flags"
  | "advanced_disable_feature_flags_on_first_load"
  | "disable_external_dependency_loading"
> = {
  advanced_disable_decide: true,
  advanced_disable_flags: true,
  advanced_disable_feature_flags: true,
  advanced_disable_feature_flags_on_first_load: true,
  disable_external_dependency_loading: true,
};

function analyticsEnabled(): boolean {
  const cfg = runtimeConfig();
  if (typeof window === "undefined" || !cfg.posthogKey) return false;
  const local = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1" || window.location.hostname === "::1";
  return !local || cfg.posthogEnableLocal;
}

function posthogClient(): Promise<PostHog> | null {
  if (!analyticsEnabled()) return null;
  if (posthogPromise) return posthogPromise;
  const cfg = runtimeConfig();
  const key = cfg.posthogKey;
  posthogPromise = import("posthog-js").then((posthog) => {
    if (!initialized) {
      initialized = true;
      posthog.default.init(key, {
        ...posthogNoRemoteConfig,
        loaded: (client) => {
          client.set_config(posthogNoRemoteConfig);
        },
        ip: false,
        capture_pageview: false,
        autocapture: false,
        disable_session_recording: true,
        disable_surveys: true,
        disable_product_tours: true,
        disable_web_experiments: true,
        disable_conversations: true,
        opt_in_site_apps: false,
        request_batching: false,
        persistence: "localStorage",
        api_host: cfg.posthogHost,
      });
    }
    return posthog.default;
  }).catch(() => {
    initialized = false;
    posthogPromise = null;
    throw new Error("posthog_unavailable");
  });
  return posthogPromise;
}

export function initAnalytics(): void {
  posthogClient()?.catch(() => {});
}

export function identify(user: { id: string; displayName?: string; email?: string; department?: string }): void {
  posthogClient()?.then((posthog) => {
    posthog.identify(user.id, {
      name: user.displayName,
      email: user.email,
      department: user.department,
    });
  }).catch(() => {});
}

export function capture(event: AnalyticsEvent, props: Record<string, unknown> = {}): void {
  if (typeof window === "undefined") return;
  posthogClient()?.then((posthog) => {
    posthog.capture(event, props);
  }).catch(() => {});
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
