# Testing Modex

This project has four independently testable surfaces:

- `backend/`: Go REST API, auth, persistence, search, deploy ingest.
- `tools/docsctl/`: documentation discovery/build/package/deploy CLI.
- `mcp/`: stdio MCP server and HTTP client wrapper.
- `frontend/`: Next.js portal.

## Go Tests

Run each Go module from its own directory:

```bash
cd backend && go test ./...
cd tools/docsctl && go test ./...
cd mcp && go test ./...
```

These are the fastest regression checks and should run on every pull request.

## Frontend Checks

```bash
cd frontend
npm ci
npm run lint
npm run build
```

`npm run lint` currently runs `next typegen && tsc --noEmit`, so it is a type
and route-contract check rather than an ESLint rule set.

## Playwright E2E

The Playwright suite lives in `frontend/e2e`. It starts the Next.js dev server
and mocks the backend API at the browser network layer, so the smoke tests do
not require PostgreSQL, MinIO, Meilisearch, or the Go API.

```bash
cd frontend
npm run e2e
```

Current smoke coverage:

- home page renders from mocked category/module data
- locale selector switches `zh-CN` to `en-US`
- mock login exposes the admin console entry

Install browsers on a new CI runner or developer machine if Playwright asks for
them:

```bash
npx playwright install chromium
```

## Recommended CI Order

1. Go tests for `backend`, `tools/docsctl`, and `mcp`.
2. Frontend `npm ci`.
3. Frontend `npm run lint`.
4. Frontend `npm run build`.
5. Frontend `npm run e2e`.

Keep E2E tests focused on user journeys. Use backend Go tests for API edge cases
and docsctl/MCP Go tests for protocol and CLI behavior.

`store.MemoryStore` is an explicit unit-test fake only. Production assembly
injects `PostgresRepository`. Set `TEST_DATABASE_URL` to run the repository
integration test, which covers request-level CRUD, publishing, analytics,
OAuth token rotation, static assets, and visibility across two repository
instances.
