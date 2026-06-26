# Branding Assets

Put deployment logos and favicon files in this directory when you want to replace the default Modex brand assets.

Supported logo filenames:

- `logo.svg`
- `logo.png`
- `logo.webp`
- `logo.jpg`
- `logo.jpeg`

For theme-specific logos, use:

- `logo-light.svg` / `logo-light.png` / `logo-light.webp` / `logo-light.jpg` / `logo-light.jpeg`
- `logo-dark.svg` / `logo-dark.png` / `logo-dark.webp` / `logo-dark.jpg` / `logo-dark.jpeg`

The frontend uses `logo-light.*` in light mode and `logo-dark.*` in dark mode. If either file is missing, it falls back to `logo.*`, then to the built-in `/logo.svg`.

Supported favicon filenames:

- `favicon.ico`
- `favicon.svg`
- `favicon.png`
- `favicon.webp`
- `favicon.jpg`
- `favicon.jpeg`

The frontend container mounts this directory to `/app/public/brand`. If the corresponding environment variables are not set, the startup script uses the first matching file above as `/brand/<filename>`.

You can also set `MODEX_PUBLIC_LOGO_URL` explicitly, for example:

```env
MODEX_PUBLIC_LOGO_URL=/brand/company-logo.svg
MODEX_PUBLIC_LOGO_LIGHT_URL=/brand/company-logo-dark-text.svg
MODEX_PUBLIC_LOGO_DARK_URL=/brand/company-logo-light-text.svg
MODEX_PUBLIC_FAVICON_URL=/brand/favicon.ico
```

Set `MODEX_PUBLIC_APP_TITLE` to customize the browser title and in-page product name:

```env
MODEX_PUBLIC_APP_TITLE=Docs Hub
```
