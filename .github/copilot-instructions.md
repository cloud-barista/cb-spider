# Copilot Instructions (for CB-Spider)

## Overview
- CB-Spider: Cloud-Barista sub-framework, unified API for multiple CSPs.
- Main runtime: REST (`api-runtime/rest-runtime`, Swagger).  
- Drivers: `cloud-control-manager/*`, `cloud-driver-libs/*`.
- Env: Ubuntu 22.04, Go 1.25, Docker 19.03+.

## Style
- Format: `gofmt`, `goimports`.
- No `panic`. Return `error`, wrap with `fmt.Errorf("...: %w", err)`.
- Always use `context.Context` for I/O, network, drivers.
- Logging: use `cb-log` (logrus). No direct print. No sensitive data.
  - cb-log: https://github.com/cloud-barista/cb-log
- Naming:  
  - Types: `*Info`, `*Req`, `*Resp`.  
  - Const: `UPPER_SNAKE_CASE`.  
  - Vars: short, standard abbreviations only.  
- Dependencies: minimal, in `go.mod`, no license conflicts.

## Layers
- **REST runtime**: validate input → call common runtime → map response. Keep Swagger updated. Wrap CSP “Original*” responses. Consistent HTTP error mapping.
- **gRPC runtime**: deprecated.
- **Drivers**: implement full common interface. Separate model ↔ SDK mapping. Document limits. Use exponential backoff. Track CSP calls with call-log.

## Testing
- Server: Run with `AdminWeb.   

- Drivers: run the dedicated test programs provided for each CSP driver.

## Performance/Security
- Timeouts default for all external calls. Retries only if idempotent.  
- Pagination/filter for list APIs.  
- Avoid data races.  
- Secrets only via env/secret. Never log sensitive data. Validate all inputs. Enforce CSP-specific constraints.

## Docs
- Sync Swagger with API changes.  
- Keep README/wiki links valid.

## Git Rules
- Commits: Conventional Commits.  
- PRs: one feature/issue, with tests/docs.  
- Driver PRs: update resource table, document limits, add tests.

## Build/Run
- Local: `make`.  
- Container: official image.  
- Config: `setup.env`, `conf/*.conf`.
