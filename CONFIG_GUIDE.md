***

# MockingGOD V2 Configuration Guide

This document describes the structure and valid parameters for the API configuration files (e.g., `configs/api_name.json`).

## 1. File Structure Overview

The configuration file is a single JSON Object containing metadata, dynamic code settings, and a list of routes.

```json
{
  "api": "my_service",
  "mode": "ir",
  "version": "v2",
  "user_code": { ... },
  "routes": [ ... ]
}
```

---

## 2. Root Parameters

These parameters define the API's identity and operational mode.

| Key | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `api` | `string` | **Yes** | Unique identifier. Maps to the subdomain (e.g., `my_service` -> `my_service.localhost`). |
| `mode` | `string` | **Yes** | Must be `"ir"` (Intermediate Representation engine) or `"proxy"`. |
| `version` | `string` | **Yes** | Must be `"v2"` to enable User Code, Validation, and HTTP transforms. |
| `user_code` | `Object` | No | Settings for dynamic WASM execution. See [Section 3](#3-user-code-configuration). |
| `routes` | `Array` | **Yes** | List of endpoint definitions. See [Section 4](#4-route-configuration). |

---

## 3. User Code Configuration
**Key:** `user_code`

This section configures the internal WASM runtime for executing custom logic.

| Key | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `inline_source` | `string` | `null` | Raw Golang source code. **Must be escaped** (use `\n` for newlines). Requires `package main` and `//export FuncName`. |
| `source` | `string` | `null` | Path to a pre-compiled `.wasm` file (if `inline_source` is not used). |
| `timeout_ms` | `int` | `5000` | Execution timeout in milliseconds per function call. Security against infinite loops. |
| `min_instances` | `int` | `1` | Number of WASM modules to keep "hot" in the pool (prevents compilation lag). |
| `max_instances` | `int` | `10` | Maximum concurrent WASM executions allowed. |

**Example Inline Source:**
```json
"inline_source": "package main\n\n//export add\nfunc add(x, y uint64) uint64 { return x + y }\n\nfunc main() {}"
```

---

## 4. Route Configuration
**Key:** `routes` (Array of Objects)

Each object in this array represents a specific API endpoint.

| Key | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `method` | `string` | **Yes** | HTTP Method (e.g., `"GET"`, `"POST"`, `"PUT"`, `"DELETE"`). |
| `path` | `string` | **Yes** | URL path. Supports dynamic parameters (e.g., `/users/{id}`). |
| `validate` | `Object` | No | Input validation rules. |
| `transform` | `Object` | No | Data extraction and upstream logic. |
| `response` | `Object` | **Yes** | The response definition. |

### 4.1 Validation
**Key:** `validate`

Enforces rules before processing the request. Returns `400 Bad Request` if failed.

| Subsection | Description |
| :--- | :--- |
| `headers` | Map of Header names to [Rule Objects](#rule-object). |
| `query` | Map of Query Parameter names to [Rule Objects](#rule-object). |

**Rule Object:**
*   `required` (bool): Fails if the field is missing.
*   `pattern` (string): Regex string the value must match (e.g., `"^Bearer .+"`).

### 4.2 Transformation
**Key:** `transform`

Prepares the data **Context** used to build the response.

#### A. Extract (`extract`)
Pulls data from the request and saves it to variables.

| Key | Description |
| :--- | :--- |
| `from` | **Source**. Format: `source.key`.<br>Valid sources: `path`, `query`, `header`.<br>Example: `"path.id"`, `"header.Authorization"`. |
| `as` | **Variable Name**. The name used to reference this value later via `{{name}}`. |
| `type` | Optional casting. currently supports `"int"` (for math) or `"string"`. |

#### B. HTTP Calls (`http`)
Makes requests to other services.

| Key | Description |
| :--- | :--- |
| `name` | Variable name to store the JSON response (e.g., `upstream_data`). |
| `url` | Target URL. Supports templating (e.g., `"http://api.com/{{user_id}}"`). |
| `method` | HTTP Method (`"GET"`, `"POST"`). |
| `headers` | Map of headers to send. Supports templating. |
| `timeout` | Timeout in milliseconds. |

### 4.3 Response
**Key:** `response`

Defines what is sent back to the client.

| Key | Description |
| :--- | :--- |
| `status` | HTTP Status Code (e.g., `200`, `201`, `404`). |
| `headers` | Response headers map. |
| `body` | The JSON payload. Values support **Templating**. |

---

## 5. Templating & Logic System

MockingGOD uses `{{ }}` syntax to inject dynamic values into `http` configurations and `response` bodies.

### A. Variable Injection
Injects values saved during the `transform` phase.
*   **Syntax:** `{{variable_name}}`
*   **Nested JSON:** `{{upstream_data.user.email}}`

### B. User Code Execution (WASM)
Calls functions defined in `user_code`.
*   **Syntax:** `{{function_name(arg1, arg2)}}`
*   **Example:** `{{multiply(price, tax_rate)}}`
*   **Constraints:**
    1.  Arguments must resolve to **Integers**.
    2.  Go functions must be exported via `//export Name`.
    3.  Go function signatures must use `uint64`.

---

## 6. Full Example Configuration

```json
{
  "api": "store_api",
  "mode": "ir",
  "version": "v2",
  "user_code": {
    "inline_source": "package main\n\n//export calculate_total\nfunc calculateTotal(price, tax uint64) uint64 { return price + (price * tax / 100) }\n\nfunc main() {}",
    "timeout_ms": 100,
    "min_instances": 2,
    "max_instances": 5
  },
  "routes": [
    {
      "method": "POST",
      "path": "/checkout/{cart_id}",
      "validate": {
        "headers": {
          "Authorization": { "required": true, "pattern": "^Bearer .+" }
        }
      },
      "transform": {
        "extract": {
          "cart": { "from": "path.cart_id", "as": "cid" },
          "cost": { "from": "query.amount", "as": "amt", "type": "int" },
          "tax":  { "from": "query.tax_rate", "as": "tax", "type": "int" }
        },
        "http": [
          {
            "name": "inventory",
            "url": "https://internal-inventory.local/check/{{cid}}",
            "method": "GET",
            "timeout": 200
          }
        ]
      },
      "response": {
        "status": 200,
        "body": {
          "cart_id": "{{cid}}",
          "stock_status": "{{inventory.status}}",
          "final_amount": "{{calculate_total(amt, tax)}}"
        }
      }
    }
  ]
}
```
