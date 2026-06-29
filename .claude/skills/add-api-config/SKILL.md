---
name: add-api-config
description: Scaffold a new MIMICORE API config JSON in configs/ following the v2 schema. Use when the user wants to add a new mocked/proxied API, a new route, validation, rate limiting, or WASM user_code.
---

# Add an API config

Create a new v2 API configuration file under `configs/<api>.json`. The full schema reference is `@CONFIG_GUIDE.md` — consult it for field details; this skill is the workflow.

## Steps

1. **Gather intent.** Confirm: the `api` id (lowercase, used as the routing host `<api>.localhost` / `<api>.local`), `mode` (`"ir"` for mocking, `"proxy"` for Unix-socket pass-through), and the routes needed (method + path, with `{param}` for path vars). Ask only what you can't infer.

2. **Write `configs/<api>.json`** with the root structure:
   ```json
   { "api": "<id>", "mode": "ir", "version": "v2", "routes": [ ... ] }
   ```
   `version` must be `"v2"`. Add `rate_limit` (`requests_per_second`, `burst`) and `user_code` only if requested.

3. **Per route**, include only the blocks that apply:
   - `validate` — `headers`/`query` (`required` bool, `pattern` regex) and `body.schema` (JSON Schema). Failures return `400`.
   - `delay` — `fixed_ms` + `jitter_ms` to simulate latency.
   - `transform.extract` — pull `path.x` / `query.x` / `header.x` / `body` into named vars (`from` → `as`).
   - `transform.http` — upstream calls (array runs concurrently); results land in vars named by `name`.
   - `response` — `status`, `headers`, `body` with `{{var}}` / `{{nested.field}}` / `{{wasm_func(var)}}` templating.

4. **WASM `user_code`** (only if dynamic logic is needed): `inline_source` is raw Go with `package main` and a `func main(){}`, exported via `//export name`. Use Pattern A `func(x,y uint64) uint64` for integer math, Pattern B `func(ptr *byte, size uint32) uint64` for string/JSON (see CONFIG_GUIDE.md §3 for the pointer-packing boilerplate). Strings in JSON must escape `\n` and `\"`.

5. **Validate the JSON** (`jq . configs/<api>.json`). Remind the user that `configs/` is gitignored and is hot-reloaded by the running gateway.

6. Offer to verify with `/run-gateway` using a `curl -H "Host: <api>.local"` against a route.
