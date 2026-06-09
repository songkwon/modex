// Analytics helper. PostHog integration is wired here behind env config so the
// rest of the app can capture events without importing posthog directly.
//
// To enable PostHog: `npm install posthog-js`, set NEXT_PUBLIC_POSTHOG_KEY and
// (optionally) NEXT_PUBLIC_POSTHOG_HOST, then uncomment the posthog-js calls.

export type AnalyticsEvent =
  | "docs_home_view"
  | "docs_module_click"
  | "docs_module_info_open"
  | "docs_page_view"
  | "docs_search"
  | "docs_search_result_click"
  | "docs_version_switch"
  | "docs_source_click"
  | "docs_mcp_search"
  | "docs_mcp_get_page";

let initialized = false;

export function initAnalytics(): void {
  if (initialized || typeof window === "undefined") return;
  initialized = true;
  const key = process.env.NEXT_PUBLIC_POSTHOG_KEY;
  if (!key) return;
  // import posthog from "posthog-js";
  // posthog.init(key, { api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://app.posthog.com" });
}

export function identify(user: { id: string; displayName?: string; email?: string; department?: string; groups?: string[] }): void {
  if (typeof window === "undefined" || !process.env.NEXT_PUBLIC_POSTHOG_KEY) return;
  // posthog.identify(user.id, {
  //   name: user.displayName,
  //   email: user.email,
  //   department: user.department,
  //   groups: user.groups,
  // });
}

export function capture(event: AnalyticsEvent, props: Record<string, unknown> = {}): void {
  if (typeof window === "undefined") return;
  if (process.env.NEXT_PUBLIC_POSTHOG_KEY) {
    // posthog.capture(event, props);
  }
  if (process.env.NODE_ENV !== "production") {
    // Helpful during MVP development before PostHog is enabled.
    console.debug("[analytics]", event, props);
  }
}

// sessionId returns a per-browser-session identifier used to compute UV/PV on
// the backend until real auth sessions are wired through.
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
