# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

MIMICORE (formerly MockingGOD) is a declarative API gateway / API-mocking platform in Go. Requests are routed to a virtual host by the HTTP `Host` header and handled by an in-memory IR (intermediate representation), with optional user logic compiled to WASM at runtime.

## Build, run, test

- `go mod tidy` — sync dependencies.
- `go run cmd/gateway/*.go` — run the gateway standalone (the entrypoint is split across several files in `cmd/gateway/`, so glob with `*.go`, not just `main.go`). Listens on `:8080` (API) and `:9090` (Prometheus metrics).
- `./gateway <config-dir>` — built binary; config dir defaults to `configs/`.
- `docker-compose up -d` — full stack: gateway + MinIO + Prometheus + Grafana.
- `go test ./... -v` — run tests.

## Non-obvious gotchas

- **Host-based routing.** An API config with `"api": "shop_api"` is only reachable via `Host: shop_api.localhost` or `Host: shop_api.local`. A plain `curl localhost:8080/...` with no `Host` header will not match — always pass `-H "Host: <api>.local"`.
- **`configs/` is gitignored** (the whole directory, plus `*_test.go`). Config files are mounted/loaded at runtime, not committed. Don't expect them in the repo.
- **Tests are not on this branch.** `*_test.go` is gitignored here; test files live on separate `test/*` branches.
- **MinIO startup.** With `MINIO_ENDPOINT` / `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` set, the gateway retries MinIO for up to 30s and then crashes if unreachable. With them unset it runs in single-instance mode (no S3 WASM cache) and logs warnings.
- **Hot-reload.** A file watcher on the config dir rebuilds the handler registry and swaps it in with zero downtime — changing a config file at runtime is expected to take effect without a restart.

## WASM / user code flow

`user_code.inline_source` (raw Go) is compiled by **TinyGo 0.33.0** to WebAssembly, hash-cached (L1 in-memory → L2 S3/MinIO), and executed in a `wazero` instance pool. TinyGo must be on `PATH` for dynamic compilation; the Docker image bundles it. Memory helpers (`_guest_alloc`) are injected automatically.

## Configuration

`@CONFIG_GUIDE.md` is the definitive reference for the v2 JSON config schema (routes, validation, `transform`/`http` chaining, rate limiting, `user_code`). Read it before editing or generating configs.

## Layout

- `cmd/gateway/` — entrypoint, hot-reload orchestration, WASM GC.
- `internal/engine/` — router, HTTP handling, validation, response templating, proxy.
- `internal/services/` — `compiler/` (TinyGo + hash cache), `storage/` (MinIO/S3), `wasm/` (wazero pool).
- `internal/ir/`, `internal/adapters/` (v1/v2 config parsing), `internal/config/`, `internal/middleware/` (rate limit, CORS, metrics).

## Conventions

- Format with `gofmt` before committing (no other linter is configured).
- **Branching (GitFlow):** `feature/*` → `development` → `release/dev` → `main`. CI (`.github/workflows/docker-publish.yml`) runs on `main` and `v*.*.*` tags and publishes `ghcr.io/axis0047/mimicore`.
