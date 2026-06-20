# Contributing to Modex

Thank you for improving Modex. By participating, you agree to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Development setup

Requirements: Go 1.23, Node.js 20+, Docker with Compose, and Chromium for the
Playwright suite.

```bash
cp deploy/.env.example deploy/.env
docker compose -f deploy/docker-compose.yml up --build
```

Run the fast checks before opening a pull request:

```bash
cd backend && go test ./... && go vet ./...
cd ../mcp && go test ./... && go vet ./...
cd ../tools/docsctl && go test ./... && go vet ./...
cd ../../frontend && npm ci && npm run i18n:check && npm run lint && npm run build
```

Use `npm run e2e` for user-facing changes. Add focused tests for behavior you
change. PostgreSQL integration tests require `TEST_DATABASE_URL`.

## Pull requests

- Keep changes scoped and explain user-visible behavior and migration impact.
- Do not commit credentials, generated `.env` files, or local configuration.
- Add Chinese source messages and English translations together. Run
  `npm run i18n:extract` after changing UI copy.
- Update `CHANGELOG.md` for notable behavior, security, or compatibility changes.
- Use Conventional Commit style where practical, for example `fix:`, `feat:`,
  `docs:`, or `refactor:`.

Report security issues privately as described in [SECURITY.md](SECURITY.md).
