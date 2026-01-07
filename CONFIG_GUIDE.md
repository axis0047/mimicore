---
# MockingGOD V2 Configuration Guide

This document describes the structure and parameters for the V2 API configuration files.

## 1. Root Configuration

| Key | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `api` | `string` | **Yes** | Unique ID. Maps to Host header (e.g., `api.localhost`). |
| `mode` | `string` | **Yes** | `"ir"` (Mocking Engine) or `"proxy"`. |
| `version` | `string` | **Yes** | Must be `"v2"`. |
| `user_code` | `Object` | No | Dynamic code settings. |
| `routes` | `Array` | **Yes** | List of endpoint definitions. |

## 2. User Code (WASM) Configuration
**Key:** `user_code`

MockingGOD automatically handles memory allocation between Go and WASM. You simply write the logic.

| Key | Description |
| :--- | :--- |
| `inline_source` | Raw Golang code. **Must be escaped** (`\n`). Requires `package main`. |
| `timeout_ms` | Max execution time per function call. |
| `min_instances` | Number of hot WASM instances to keep in the pool. |

### How to write User Code
1.  **Integers:** Use `func add(x, y uint64) uint64`. Call via `{{add(var1, var2)}}`.
2.  **Strings/JSON:** Use `func process(ptr *byte, size uint32) uint64`. Call via `{{process(var_json)}}`.
    *   *Note:* The system automatically injects `_guest_alloc` helpers. You do not need to write `malloc` yourself anymore.

## 3. Route Configuration

| Key | Description |
| :--- | :--- |
| `validate` | Input validation rules. |
| `transform` | Data extraction and Upstream calls. |
| `delay` | Network latency simulation. |
| `response` | The response definition. |

### 3.1 Validation
**Key:** `validate`

*   **Headers/Query:** Regex pattern matching.
*   **Body:** **Full JSON Schema** support.

```json
"validate": {
  "headers": { "Authorization": { "required": true, "pattern": "^Bearer .+" } },
  "body": {
    "schema": {
      "type": "object",
      "required": ["age"],
      "properties": { "age": { "type": "integer", "minimum": 18 } }
    }
  }
}
```

### 3.2 Latency Simulation
**Key:** `delay`

Simulate network issues before sending the response.

*   `fixed_ms`: Base delay in milliseconds.
*   `jitter_ms`: Random additional delay (0 to `jitter_ms`).

```json
"delay": { "fixed_ms": 200, "jitter_ms": 100 }
```

### 3.3 Transformation
**Key:** `transform`

*   **Extract:** Pull data from `path`, `query`, `header`, or `body`.
*   **HTTP:** Make upstream calls. **Note:** All HTTP calls in this list run in **PARALLEL**.

```json
"transform": {
  "extract": { "my_input": { "from": "body", "as": "input_json" } },
  "http": [
    { "name": "service_a", "url": "http://a.com", "method": "GET" },
    { "name": "service_b", "url": "http://b.com", "method": "GET" }
  ]
}
```

### 3.4 Response
**Key:** `response`

Use `{{ }}` templates to inject data from extraction, upstream calls, or WASM results.

```json
"response": {
  "status": 200,
  "body": {
    "data_a": "{{service_a.data}}",
    "calculated": "{{my_wasm_func(input_json)}}"
  }
}
```

---

## 4. Full Example: Advanced Logic

This example validates a user via JSON schema, extracts the body, simulates network lag, and processes the JSON using inline Go code.

```json
{
  "api": "advanced_api",
  "mode": "ir",
  "version": "v2",
  "user_code": {
    "timeout_ms": 500,
    "inline_source": "package main\nimport (\n\t\"encoding/json\"\n\t\"unsafe\"\n)\n\ntype User struct { Name string `json:\"name\"` }\ntype Resp struct { Msg string `json:\"msg\"` }\n\n//export greet\nfunc greet(ptr *byte, size uint32) uint64 {\n\t// Boilerplate to read string\n\tbytes := unsafe.Slice(ptr, size)\n\tvar u User\n\tjson.Unmarshal(bytes, &u)\n\t\n\t// Logic\n\tr := Resp{Msg: \"Hello \" + u.Name}\n\tout, _ := json.Marshal(r)\n\t\n\t// Boilerplate to return string\n\tlen := uint32(len(out))\n\tptrOut := uintptr(unsafe.Pointer(&out[0]))\n\treturn (uint64(ptrOut) << 32) | uint64(len)\n}\nfunc main() {}"
  },
  "routes": [
    {
      "method": "POST",
      "path": "/greet",
      "validate": {
        "body": {
          "schema": {
            "type": "object",
            "required": ["name"],
            "properties": { "name": { "type": "string" } }
          }
        }
      },
      "delay": { "fixed_ms": 150 },
      "transform": {
        "extract": { "raw": { "from": "body", "as": "input_json" } }
      },
      "response": {
        "status": 200,
        "body": { "result": "{{greet(input_json)}}" }
      }
    }
  ]
}
```
---
