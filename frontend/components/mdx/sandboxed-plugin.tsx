"use client";

import { useEffect, useMemo, useState } from "react";
import { useI18n } from "@/lib/i18n";

// Renders an admin-imported JSX plugin inside an isolated iframe. The iframe is
// sandboxed with allow-scripts but WITHOUT allow-same-origin → it gets an opaque
// origin and cannot read the parent's cookies, DOM, localStorage or call modex
// APIs as the user. React/ReactDOM/Babel are self-hosted under /plugin-runtime
// and run independently of the host app's React. This is the core XSS containment.

// Base64 (UTF-8 safe) so the embedded code/props contain only [A-Za-z0-9+/=]
// and can never break out of their host <script> element — no <, >, quotes or
// "</script>" can appear, eliminating the injection surface entirely.
function b64(s: string): string {
  if (typeof window === "undefined") return "";
  return btoa(unescape(encodeURIComponent(s)));
}

export function SandboxedPlugin({
  code,
  props,
  minHeight = 32
}: {
  code: string;
  props?: Record<string, unknown>;
  minHeight?: number;
}) {
  const { t } = useI18n();
  const [mounted, setMounted] = useState(false);
  const [height, setHeight] = useState(minHeight);
  const [error, setError] = useState<string | null>(null);
  const nonce = useMemo(() => Math.random().toString(36).slice(2), []);

  useEffect(() => setMounted(true), []);

  const srcDoc = useMemo(() => {
    if (!mounted) return "";
    const origin = window.location.origin;
    const codeB64 = b64(code);
    const propsB64 = b64(JSON.stringify(props || {}));
    return `<!doctype html><html><head><meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'unsafe-inline' 'unsafe-eval' ${origin}; style-src 'unsafe-inline'; img-src * data: blob:; media-src * data: blob:; font-src * data:; connect-src *">
<style>html,body{margin:0;padding:0}body{font:14px/1.6 system-ui,-apple-system,'Segoe UI',Roboto,sans-serif;color:#1d2733}#__root{padding:2px}</style>
</head><body>
<div id="__root"></div>
<script type="application/json" id="__code">${codeB64}</script>
<script type="application/json" id="__props">${propsB64}</script>
<script>function report(t,p){parent.postMessage({__modexPlugin:'${nonce}',type:t,payload:p},'*');}
window.addEventListener('error',function(e){report('error',String(e.message));});
function sendHeight(){report('height',document.documentElement.scrollHeight);}
window.addEventListener('load',sendHeight);try{new ResizeObserver(sendHeight).observe(document.documentElement);}catch(_){}
setTimeout(sendHeight,250);setTimeout(sendHeight,800);
function dec(id){return decodeURIComponent(escape(atob(document.getElementById(id).textContent||'')));}</script>
<script src="${origin}/plugin-runtime/react.production.min.js"></script>
<script src="${origin}/plugin-runtime/react-dom.production.min.js"></script>
<script src="${origin}/plugin-runtime/babel.min.js"></script>
<script>(function(){
  try {
    var code = dec('__code');
    var props = JSON.parse(dec('__props') || '{}');
    var out = Babel.transform(code, { presets: ['react'] }).code;
    var Plugin = (new Function(out + '\\n;return typeof Plugin!=="undefined"?Plugin:undefined;'))();
    if (!Plugin) throw new Error('插件需定义 function Plugin(props){ … }');
    ReactDOM.createRoot(document.getElementById('__root')).render(React.createElement(Plugin, props));
  } catch (e) { report('error', String((e && e.message) || e)); }
})();</script>
</body></html>`;
  }, [mounted, code, props, nonce]);

  useEffect(() => {
    function onMsg(e: MessageEvent) {
      const d = e.data;
      if (!d || d.__modexPlugin !== nonce) return;
      if (d.type === "height" && typeof d.payload === "number") setHeight(Math.max(minHeight, d.payload));
      else if (d.type === "error") setError(String(d.payload));
    }
    window.addEventListener("message", onMsg);
    return () => window.removeEventListener("message", onMsg);
  }, [nonce, minHeight]);

  if (!mounted) return <div className="mdx-plugin mdx-plugin--loading" />;
  if (error) {
    return (
      <div className="mdx-plugin mdx-plugin--error">
        <span className="mdx-plugin__tag">{t("component.mdx.sandboxedPlugin.third_party_plugin_error")}</span>
        <pre>{error}</pre>
      </div>
    );
  }
  return (
    <iframe
      className="mdx-plugin__frame"
      sandbox="allow-scripts allow-popups allow-popups-to-escape-sandbox"
      srcDoc={srcDoc}
      style={{ width: "100%", height, border: 0, display: "block" }}
      title="plugin"
    />
  );
}
