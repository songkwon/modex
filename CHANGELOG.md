# Changelog

This project follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and uses semantic version tags.

## Unreleased

### Security

- Added verified OIDC ID tokens, nonce validation, PKCE, persistent Redis
  sessions, request limits, and hardened HTTP server timeouts.

### Changed

- Upgraded the project toolchain to Go 1.23.
- Split the API and in-memory store into responsibility-focused files.
- Completed English catalog coverage for existing frontend UI copy.
- Added checksums, SBOMs, signatures, provenance, and container publication to
  the release workflow.
