"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Loader2, Send } from "lucide-react";
import { usePluginConfig, pluginEnabled, pluginValue } from "./mdx-config";
import { useI18n } from "@/lib/i18n";

const METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE"];

type Resp = { status: number; ms: number; text: string } | { error: string };

function asText(v: unknown): string {
  if (v == null) return "";
  if (typeof v === "string") return v;
  try {
    return JSON.stringify(v, null, 2);
  } catch {
    return String(v);
  }
}

// Interactive "try it" console. method/url/headers/body are editable; Send fires
// a client fetch and shows status, latency and the (pretty-printed) response.
export function ApiPlayground({
  method = "GET",
  url = "",
  baseUrl = "",
  title,
  headers,
  body
}: {
  method?: string;
  url?: string;
  baseUrl?: string;
  title?: string;
  headers?: unknown;
  body?: unknown;
}) {
  const { t } = useI18n();
  const cfg = usePluginConfig();
  const [m, setM] = useState(method.toUpperCase());
  const [u, setU] = useState(url);
  const [headersText, setHeadersText] = useState(asText(headers));
  const [bodyText, setBodyText] = useState(asText(body));
  const [resp, setResp] = useState<Resp | null>(null);
  const [loading, setLoading] = useState(false);

  const fullUrl = useMemo(() => {
    if (/^https?:\/\//.test(u)) return u;
    const b = baseUrl.replace(/\/+$/, "");
    return b ? `${b}/${u.replace(/^\/+/, "")}` : u;
  }, [u, baseUrl]);

  if (!pluginEnabled(cfg, "openapi")) return null;

  async function send() {
    setLoading(true);
    setResp(null);
    const t0 = performance.now();
    try {
      let hdrs: Record<string, string> = {};
      if (headersText.trim()) {
        try {
          hdrs = JSON.parse(headersText);
        } catch {
          throw new Error(t("component.mdx.apiPlayground.request_headers_are_not_valid_json"));
        }
      }
      const init: RequestInit = { method: m, headers: hdrs };
      if (m !== "GET" && m !== "HEAD" && bodyText.trim()) init.body = bodyText;
      const r = await fetch(fullUrl, init);
      const text = await r.text();
      let pretty = text;
      try {
        pretty = JSON.stringify(JSON.parse(text), null, 2);
      } catch {
        /* keep raw */
      }
      setResp({ status: r.status, ms: Math.round(performance.now() - t0), text: pretty });
    } catch (e) {
      setResp({ error: e instanceof Error ? e.message : t("component.mdx.apiPlayground.request_failed_2") });
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mdx-apiplay">
      {title ? <div className="mdx-apiplay__title">{title}</div> : null}
      <div className="mdx-apiplay__bar">
        <select className="mdx-apiplay__method" value={m} onChange={(e) => setM(e.target.value)}>
          {METHODS.map((x) => (
            <option key={x} value={x}>{x}</option>
          ))}
        </select>
        <input className="mdx-apiplay__url" value={u} placeholder="https://api.example.com/v1/resource" onChange={(e) => setU(e.target.value)} />
        <button className="mdx-apiplay__send" onClick={send} disabled={loading || !fullUrl}>
          {loading ? <Loader2 size={14} className="ds-spin" /> : <Send size={14} />} {t("component.docChatPanel.send")}
        </button>
      </div>
      <details className="mdx-apiplay__opt">
        <summary>{t("component.mdx.apiPlayground.request_headers_request_body")}</summary>
        <label className="mdx-apiplay__lbl">Headers（JSON）</label>
        <textarea className="mdx-apiplay__ta" value={headersText} placeholder='{"Authorization": "Bearer …"}' onChange={(e) => setHeadersText(e.target.value)} />
        {m !== "GET" && m !== "HEAD" ? (
          <>
            <label className="mdx-apiplay__lbl">Body</label>
            <textarea className="mdx-apiplay__ta" value={bodyText} placeholder='{"key": "value"}' onChange={(e) => setBodyText(e.target.value)} />
          </>
        ) : null}
      </details>
      {resp ? (
        <div className="mdx-apiplay__resp">
          {"error" in resp ? (
            <div className="mdx-apiplay__status mdx-apiplay__status--err">{t("component.mdx.apiPlayground.request_failed")}{resp.error}</div>
          ) : (
            <>
              <div className={`mdx-apiplay__status${resp.status >= 400 ? " mdx-apiplay__status--err" : ""}`}>
                {resp.status} · {resp.ms}ms
              </div>
              <pre className="mdx-apiplay__body">{resp.text}</pre>
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}

// Passive Mintlify-compat containers for hand-authored samples.
export function RequestExample({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  return (
    <div className="mdx-apix mdx-apix--req">
      <div className="mdx-apix__label">{t("component.mdx.apiPlayground.request_example")}</div>
      {children}
    </div>
  );
}
export function ResponseExample({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  return (
    <div className="mdx-apix mdx-apix--res">
      <div className="mdx-apix__label">{t("component.mdx.apiPlayground.response_example")}</div>
      {children}
    </div>
  );
}

// ---- OpenAPI operation reference --------------------------------------------

const specCache = new Map<string, Promise<any>>();
function loadSpec(url: string): Promise<any> {
  if (!specCache.has(url)) {
    specCache.set(
      url,
      fetch(url).then((r) => {
        if (!r.ok) throw new Error(`spec ${r.status}`);
        return r.json();
      })
    );
  }
  return specCache.get(url)!;
}

type Param = { name?: string; in?: string; required?: boolean; schema?: { type?: string }; description?: string };

// Renders one OpenAPI operation: summary, parameters, response codes, and a
// prefilled <ApiPlayground>. JSON specs only; $ref schemas are shown shallowly.
export function OpenApi({ spec, operation, title }: { spec?: string; operation?: string; title?: string }) {
  const { t } = useI18n();
  const cfg = usePluginConfig();
  const specUrl = spec || pluginValue(cfg, "openapi", "default_spec_url") || "";
  const [data, setData] = useState<any>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!pluginEnabled(cfg, "openapi")) return;
    if (!specUrl) {
      setErr(t("component.mdx.apiPlayground.openapi_specification_url_spec_property_or_plugin_default"));
      return;
    }
    let cancelled = false;
    loadSpec(specUrl)
      .then((d) => !cancelled && setData(d))
      .catch((e) => !cancelled && setErr(String(e)));
    return () => {
      cancelled = true;
    };
  }, [cfg, specUrl]);

  if (!pluginEnabled(cfg, "openapi")) return null;

  const [rawMethod, rawPath] = (operation || "").trim().split(/\s+/);
  const method = (rawMethod || "get").toLowerCase();
  const path = rawPath || "";

  if (err) return <div className="mdx-apiplay__status mdx-apiplay__status--err">{err}</div>;
  if (!data) return <div className="mdx-openapi__loading">{t("component.mdx.apiPlayground.loading_openapi_specification")}</div>;

  const op = data?.paths?.[path]?.[method];
  if (!op) {
    return <div className="mdx-apiplay__status mdx-apiplay__status--err">{t("component.mdx.apiPlayground.operation_not_found_in_specification")}{method.toUpperCase()} {path}</div>;
  }
  const server = (data.servers?.[0]?.url || "").replace(/\/+$/, "");
  const params: Param[] = op.parameters || [];
  const responses: Record<string, { description?: string }> = op.responses || {};

  return (
    <div className="mdx-openapi">
      <div className="mdx-openapi__head">
        <span className={`mdx-openapi__method mdx-openapi__method--${method}`}>{method.toUpperCase()}</span>
        <code className="mdx-openapi__path">{path}</code>
      </div>
      {title || op.summary ? <p className="mdx-openapi__summary">{title || op.summary}</p> : null}

      {params.length ? (
        <div className="mdx-openapi__section">
          <h4>{t("component.mdx.apiPlayground.parameters")}</h4>
          {params.map((p, i) => (
            <div className="mdx-field" key={i}>
              <div className="mdx-field__head">
                {p.name ? <code className="mdx-field__name">{p.name}</code> : null}
                {p.in ? <span className="mdx-field__type">{p.in}</span> : null}
                {p.schema?.type ? <span className="mdx-field__type">{p.schema.type}</span> : null}
                {p.required ? <span className="mdx-field__req">{t("component.mdx.apiPlayground.required")}</span> : null}
              </div>
              {p.description ? <div className="mdx-field__body">{p.description}</div> : null}
            </div>
          ))}
        </div>
      ) : null}

      {Object.keys(responses).length ? (
        <div className="mdx-openapi__section">
          <h4>{t("component.mdx.apiPlayground.response")}</h4>
          {Object.entries(responses).map(([code, r]) => (
            <div className="mdx-field" key={code}>
              <div className="mdx-field__head">
                <code className="mdx-field__name">{code}</code>
                {r.description ? <span className="mdx-field__body" style={{ marginTop: 0 }}>{r.description}</span> : null}
              </div>
            </div>
          ))}
        </div>
      ) : null}

      <ApiPlayground method={method} url={`${server}${path}`} title={t("component.mdx.apiPlayground.debug")} />
    </div>
  );
}
