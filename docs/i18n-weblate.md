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
- The language switcher lives in the signed-in user menu and switches locale
  without changing routes.

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
npm run i18n:extract
npm run i18n:check
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

## Migration compatibility

New and actively maintained screens should use semantic keys through
`useI18n().t(...)`. Existing UI literals are covered by deterministic
`legacy.<content-hash>` entries generated from the TypeScript syntax tree. The
provider translates those text nodes and accessibility attributes at runtime,
so switching locales covers the full existing interface while components move
to semantic keys incrementally.

`npm run i18n:check` fails when a generated entry is untranslated or when its
placeholders differ between locales. Code examples are deliberately excluded so
translation never changes executable snippets.
