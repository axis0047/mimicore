---
name: run-gateway
description: Build/run the MIMICORE gateway locally and exercise an endpoint with the correct Host header to verify a change. Use when asked to run the gateway, test a config, or confirm a route works.
---

# Run the gateway and hit an endpoint

The gateway routes by `Host` header, so a plain `curl localhost:8080/...` will not match any API. Always send `Host: <api>.local`.

## Steps

1. **Decide single-instance vs full stack:**
   - Quick local run (no S3 WASM cache): `go run cmd/gateway/*.go` — serves `:8080` (API), `:9090` (metrics). Run it in the background and tail the log.
   - Full stack (MinIO + Prometheus + Grafana): `docker-compose up -d`. Use this when testing WASM caching or multi-instance behavior.
   - If `MINIO_*` env vars are set but MinIO is unreachable, the gateway retries ~30s then exits — either start MinIO or unset those vars for a single-instance run.

2. **Confirm it's up** before sending traffic — wait for the listening log line (or poll `:9090/metrics`). Don't fire curls at a port that isn't ready yet.

3. **Hit a route** with the API's host. Replace `<api>` with the config's `api` id and match the route's method/path:
   ```bash
   curl -v -H "Host: <api>.local" http://localhost:8080/<path>
   # POST with body + validation headers:
   curl -v -X POST -H "Host: <api>.local" \
     -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
     -d '{...}' "http://localhost:8080/<path>?<query>"
   ```

4. **Report** the status code and body, and tie it back to the expected behavior (e.g. `400` on a schema-validation failure, `429` once the rate-limit burst is exhausted).

5. Stop the background process / `docker-compose down` when done if you started it.
