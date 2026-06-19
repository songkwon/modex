# Internationalization and Weblate

Modex uses simple JSON message catalogs so translation tools can work without
understanding the React component tree.

## Catalog Files

Source files:

- `frontend/messages/zh-CN.json`
- `frontend/messages/en-US.json`

`zh-CN` is the source language. Add new keys there first, then mirror the same
keys in every other locale file.

Keys are flat dot-separated strings:

```json
{
  "home.title": "文档中心",
  "search.placeholder": "搜索文档，或向 AI 提问…"
}
```

Use `{{name}}` placeholders for runtime values:

```json
{
  "home.loginFailed": "登录失败：{{error}}"
}
```

In React components:

```tsx
const { t } = useI18n();
t("home.loginFailed", { error: loginError });
```

## Runtime Behavior

The frontend wraps the app in `I18nProvider` from `frontend/lib/i18n.tsx`.

- Default locale: browser language when supported, otherwise `zh-CN`.
- User choice is stored in `localStorage` as `modex_locale`.
- The top bar language selector switches locale without changing routes.

This keeps the current URL model stable while still allowing Weblate-managed
catalogs. If route-prefixed locales are needed later, the catalog format can stay
the same.

## Weblate Setup Notes

Recommended component settings:

- File mask: `frontend/messages/*.json`
- Base file: `frontend/messages/zh-CN.json`
- Source language: `zh-CN`
- Translation files: one JSON file per locale, for example `en-US.json`
- JSON format: flat key-value catalog

Before importing into Weblate, make sure every locale has the same key set:

```bash
cd frontend
npm run lint
```

TypeScript imports the JSON catalogs, so missing or malformed JSON is caught by
the frontend checks.

## Contributor Rules

- Do not hard-code new user-facing strings in migrated components.
- Prefer clear, stable keys such as `user.logout`, not sentence fragments.
- Keep placeholders identical across locales.
- Do not translate product names such as `Modex`, `MCP`, `docsctl`, or `Deploy Token`.
- Avoid putting HTML in message strings; compose markup in React and translate
  the visible text only.

## Current Migration Scope

The shell, home page, top-bar search/AI controls, global search palette, main
document search box, and user menu are migrated. Admin pages still contain some
inline Chinese copy and can be migrated incrementally as those screens stabilize.
