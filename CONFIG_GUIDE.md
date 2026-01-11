***

# MockingGOD V2 Configuration Guide

This document is the definitive reference for configuring APIs in MockingGOD. The configuration file uses JSON format.

## 1. Root Structure

Every configuration file represents one API service (Virtual Host).

```json
{
  "api": "payment_service",
  "mode": "ir",
  "version": "v2",
  "rate_limit": { ... },
  "user_code": { ... },
  "routes": [ ... ]
}
```

| Key | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `api` | `string` | **Yes** | The unique ID. Traffic is routed via the Host header: `{api}.localhost` or `{api}.local`. |
| `mode` | `string` | **Yes** | `"ir"` (Mocking Engine) or `"proxy"` (Unix Socket pass-through). |
| `version` | `string` | **Yes** | Must be `"v2"` to use features listed below. |
| `rate_limit`| `Object` | No | Traffic control settings (Token Bucket). |
| `user_code` | `Object` | No | Dynamic WASM compilation settings. |
| `routes` | `Array` | **Yes** | List of endpoint logic. |

---

## 2. Traffic Control (Rate Limiting)
**Key:** `rate_limit`

Protects the gateway from abuse using a local **Token Bucket** algorithm.

| Key | Type | Description |
| :--- | :--- | :--- |
| `requests_per_second` | `float` | The refill rate of tokens. |
| `burst` | `int` | Maximum tokens allowed at once. Allows sudden spikes. |

**Example:**
```json
"rate_limit": { "requests_per_second": 50, "burst": 100 }
```
*Effect: Allows 100 immediate requests, then throttles to 50 req/s. Returns `429 Too Many Requests` when exceeded.*

---

## 3. Dynamic User Code (WASM)
**Key:** `user_code`

MockingGOD includes a built-in compiler (TinyGo). It accepts raw Go code strings, compiles them to WebAssembly, caches them in S3/MinIO, and runs them in a sandboxed pool.

**Memory Management:** The system automatically injects memory helpers (`_guest_alloc`). You do not need to write `malloc` manually.

| Key | Description |
| :--- | :--- |
| `inline_source` | Raw Golang code. **Must be escaped** (`\n` for newlines, `\"` for quotes). Requires `package main`. |
| `timeout_ms` | Max execution time per function. Prevents infinite loops. |
| `min_instances` | Number of "hot" WASM instances to keep ready in the pool. |
| `max_instances` | Hard limit on concurrent WASM executions. |

### 3.1 Writing Logic (Go)

**Pattern A: Integer Math (Fastest)**
Signature: `func name(x, y uint64) uint64`
Config Usage: `{{name(var1, var2)}}`

**Pattern B: String/JSON Processing (Powerful)**
Signature: `func name(ptr *byte, size uint32) uint64`
Config Usage: `{{name(json_var)}}`

*Boilerplate for Pattern B:*
```go
import ("encoding/json"; "unsafe")

//export my_func
func my_func(ptr *byte, size uint32) uint64 {
    // 1. Read Input
    inBytes := unsafe.Slice(ptr, size)
    // ... process data ...
    
    // 2. Write Output
    outBytes := []byte("result string or json")
    
    // 3. Return Pointer/Length packed into uint64
    len := uint32(len(outBytes))
    ptrOut := uintptr(unsafe.Pointer(&outBytes[0]))
    return (uint64(ptrOut) << 32) | uint64(len)
}
```

---

## 4. Route Definition
**Key:** `routes` (Array)

### 4.1 Matching
| Key | Description |
| :--- | :--- |
| `method` | HTTP Method (`GET`, `POST`, `PUT`, `DELETE`, etc). |
| `path` | URL Path. Use `{param}` for dynamic variables. Example: `/users/{id}/details`. |

### 4.2 Validation (`validate`)
Returns `400 Bad Request` if rules fail.

*   **Headers / Query:** Key-value pairs. Supports `required` (bool) and `pattern` (Regex string).
*   **Body:** Supports full **JSON Schema**.

```json
"validate": {
  "headers": { "Authorization": { "required": true, "pattern": "^Bearer .+" } },
  "body": {
    "schema": {
      "type": "object",
      "required": ["sku", "qty"],
      "properties": { "qty": { "type": "integer", "minimum": 1 } }
    }
  }
}
```

### 4.3 Network Simulation (`delay`)
Simulates network lag before sending the response.

*   `fixed_ms`: Base latency.
*   `jitter_ms`: Random added latency (0 to N milliseconds).

### 4.4 Transformation (`transform`)
Prepares the **Data Context** for the response.

**A. Extract (`extract`)**
Pulls data into variables.
*   `from`: Source (`path.x`, `query.x`, `header.x`, or `body` for full JSON).
*   `as`: Variable name used in templates.

**B. HTTP Chaining (`http`)**
Calls upstream services.
*   **Parallelism:** All calls in the `http` array run **concurrently** via `errgroup`.
*   **Context:** Results are stored in variables defined by `name`.

```json
"http": [
  { "name": "user_data", "url": "http://api.com/users/{{id}}", "method": "GET" }
]
```

### 4.5 Response (`response`)
Constructs the final output using Templating `{{ }}`.

*   **Variable Injection:** `{{extracted_var}}`
*   **Nested Access:** `{{user_data.address.city}}`
*   **WASM Call:** `{{my_func(extracted_body)}}`

---

## 5. Comprehensive Example

This example configuration demonstrates a **Pre-Order Validation Service**.
1.  **Protects** itself with Rate Limiting.
2.  **Validates** the JSON body schema and Auth header.
3.  **Simulates** a 200ms DB latency.
4.  **Fetches** exchange rates from an external API.
5.  **Calculates** the final price with tax using **Custom Go Logic (WASM)**.

### `configs/api_comprehensive.json`

```json
{
  "api": "shop_api",
  "mode": "ir",
  "version": "v2",
  "rate_limit": {
    "requests_per_second": 50,
    "burst": 100
  },
  "user_code": {
    "timeout_ms": 1000,
    "min_instances": 2,
    "max_instances": 20,
    "inline_source": "package main\nimport (\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"unsafe\"\n)\n\ntype Order struct {\n\tPrice float64 `json:\"price\"`\n\tTax   float64 `json:\"tax_percent\"`\n}\n\n//export calculate_total\nfunc calculate_total(ptr *byte, size uint32) uint64 {\n\t// 1. Read JSON\n\tbytes := unsafe.Slice(ptr, size)\n\tvar o Order\n\tjson.Unmarshal(bytes, &o)\n\n\t// 2. Logic\n\ttotal := o.Price + (o.Price * o.Tax / 100.0)\n\tres := fmt.Sprintf(\"%.2f\", total)\n\n\t// 3. Return String\n\tout := []byte(res)\n\tlen := uint32(len(out))\n\tptrOut := uintptr(unsafe.Pointer(&out[0]))\n\treturn (uint64(ptrOut) << 32) | uint64(len)\n}\nfunc main() {}"
  },
  "routes": [
    {
      "method": "POST",
      "path": "/checkout/{region}",
      "validate": {
        "headers": {
          "Authorization": { "required": true, "pattern": "^Bearer sk_live_.*" }
        },
        "query": {
          "currency": { "required": true, "pattern": "^(USD|EUR)$" }
        },
        "body": {
          "schema": {
            "type": "object",
            "required": ["price", "tax_percent", "items"],
            "properties": {
              "price": { "type": "number", "minimum": 0 },
              "tax_percent": { "type": "number", "maximum": 50 },
              "items": { "type": "array" }
            }
          }
        }
      },
      "delay": {
        "fixed_ms": 200,
        "jitter_ms": 50
      },
      "transform": {
        "extract": {
          "reg": { "from": "path.region", "as": "region_code" },
          "curr": { "from": "query.currency", "as": "currency_code" },
          "raw": { "from": "body", "as": "order_body" }
        },
        "http": [
          {
            "name": "forex",
            "url": "https://api.exchangerate-api.com/v4/latest/USD",
            "method": "GET",
            "timeout": 2000
          }
        ]
      },
      "response": {
        "status": 200,
        "headers": {
          "X-Processed-By": "MockingGOD-WASM"
        },
        "body": {
          "region": "{{region_code}}",
          "currency": "{{currency_code}}",
          "exchange_rate": "{{forex.rates.EUR}}",
          "final_amount": "{{calculate_total(order_body)}}",
          "status": "approved"
        }
      }
    }
  ]
}
```

---

## 6. Testing Commands

Use these curls to verify the configuration above.

**1. Success Scenario**
Should return `200 OK` after ~200ms.
*(Note: Requires internet access for the external forex API call)*.

```bash
curl -v -X POST \
  -H "Host:shop_api.local" \
  -H "Authorization: Bearer sk_live_12345" \
  -H "Content-Type: application/json" \
  -d '{"price": 100, "tax_percent": 20, "items": ["apple"]}' \
  "http://localhost:8080/checkout/EU?currency=EUR"
```

**2. Validation Error (Schema)**
Sends invalid `price` type. Should return `400 Bad Request`.

```bash
curl -v -X POST \
  -H "Host:shop_api.local" \
  -H "Authorization: Bearer sk_live_12345" \
  -d '{"price": "free", "tax_percent": 20, "items": []}' \
  "http://localhost:8080/checkout/EU?currency=EUR"
```

**3. Validation Error (Header)**
Sends wrong Auth format. Should return `400 Bad Request`.

```bash
curl -v -X POST \
  -H "Host:shop_api.local" \
  -H "Authorization: Basic 123" \
  "http://localhost:8080/checkout/EU?currency=EUR"
```

**4. Rate Limit Test**
Spam the endpoint to trigger the Token Bucket limit (Burst 100).

```bash
for i in {1..150}; do 
  curl -s -o /dev/null -w "%{http_code}\n" \
  -H "Host:shop_api.local" \
  -H "Authorization: Bearer sk_live_X" \
  "http://localhost:8080/checkout/US?currency=USD" &
done
```
