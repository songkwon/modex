# Production upgrades and rollback

## Compatibility policy

Modex uses semantic version tags. Patch releases should be drop-in fixes. Minor
releases may add configuration and backward-compatible schema changes. A major
release may require an explicit migration described in its release notes.

The backend, frontend, MCP package, and `docsctl` from one release tag are tested
as a set. Mixing versions is unsupported unless release notes explicitly allow
it. Go 1.23 and Node.js 20 are the supported build toolchains.

## Before upgrading

1. Read the changelog and release notes.
2. Verify release checksums, Sigstore bundles, provenance, and SBOMs.
3. Back up PostgreSQL and the MinIO bucket. Record the currently deployed image
   digests and environment/configuration revisions.
4. Test the upgrade against a restored staging copy, including login, search,
   document deploy, MCP access, and rollback of a document release.

## Rolling upgrade

Stop writes from document CI when a release includes schema changes. Upgrade the
backend first, wait for `/healthz` to report healthy sessions and dependencies,
then upgrade the frontend and MCP clients. Resume document deploys after a smoke
test. Do not run mixed backend versions during schema transitions.

## Application rollback

Re-deploy the previous immutable image digest and matching frontend/MCP release.
Restore the previous configuration revision. Schema changes are forward-only;
do not run an older binary against a changed database unless the release notes
state that it is compatible. When it is not compatible, restore the PostgreSQL
and MinIO backups together to keep metadata and artifacts consistent.

Document content releases can be rolled back independently from the admin
release page; this does not roll back the Modex application itself.

## Data consistency

The API reads and writes business data directly through PostgreSQL on every
request. There is no process-local business cache, periodic autosave, or whole
store snapshot. Committed changes are immediately visible to every API
instance. Static documentation assets use MinIO when configured and otherwise
fall back to the `docs_site_file` PostgreSQL table.
