# 🌀 MockingGOD

**MockingGOD** is a high-performance, declarative API Gateway and API Mocking Platform written in Go. It uses an **Intermediate Representation (IR)** engine to decouple configuration from execution.

It is designed for scale and developer experience, featuring **zero-downtime hot-reloading**, **upstream proxying**, **parallel execution**, **distributed caching**, and **dynamic user code execution** via WebAssembly (WASM).

A variation of this project serves as the core for [mocc.dev](http://mocc.dev). This is the open-source version.

## ⚠️ Important Note
I mainly work with Python and C/C++. Go is new to me. I built this with support from AI assistants and online resources. Since this project has **user code execution** and **API chaining**, these features can be misused. If you plan to run untrusted code, use caution and double-check all code for exploits. It is recommended to run this in a restricted environment like Docker or a hardened kernel, behind a DMZ. This project is intended for demonstration and testing purposes.

## 🚀 Key Features

### Core Engine
*   **Declarative Configuration**: Define endpoints, validation, logic, and responses entirely in JSON.
*   **Zero-Downtime Hot Reload**: The system detects file changes and recompiles specific routes/WASM binaries without dropping active connections.
*   **Parallel Execution**: Transformation steps (like upstream HTTP calls) are batched and executed concurrently using `errgroup`.
*   **High Performance**: Uses `goccy/go-json` for fast parsing and a global connection pool for low-latency HTTP chaining.

### Advanced V2 Capabilities
*   **Dynamic User Code (WASM)**: Write Go code directly in your JSON. It is compiled to WASM on-the-fly and executed in a sandboxed, pooled runtime (Wazero). Supports complex JSON manipulation.
*   **Distributed Caching (S3/MinIO)**: Compiled WASM binaries are hashed and stored in S3/MinIO. This enables instant startup for clusters and prevents "thundering herd" compilation spikes.
*   **Robust Validation**: Full support for **JSON Schema** validation for request bodies, plus Regex patterns for Headers/Query params.
*   **Network Simulation**: Built-in support for **Fixed Latency** and **Jitter** to simulate real-world network conditions.
*   **Garbage Collection**: Automatically cleans up orphaned WASM binaries from object storage.

## 🛠️ Prerequisites

*   **Go 1.22+**
*   **TinyGo**: Required for the Dynamic Compiler service (to compile inline Go code to WASM).
    *   [Install TinyGo Instructions](https://tinygo.org/getting-started/install/)
*   **Docker** (Optional): Recommended for running MinIO (S3 Cache).

## 📦 Installation & Run

1.  **Clone the repository**
    ```bash
    git clone https://github.com/axis0047/mockingGOD.git
    cd mockingGOD
    ```

2.  **Install Dependencies**
    ```bash
    go mod tidy
    ```

3.  **Start the Gateway** (Standard Mode)
    ```bash
    go run cmd/gateway/*.go
    ```
    *Server listens on port `:8080`.*

## 🗄️ S3/MinIO Integration (Recommended)

To enable **Distributed Caching** (skipping compilation on restart) and **Garbage Collection**, run a MinIO instance.

1.  **Start MinIO in Docker:**
    ```bash
    docker run -d -p 9000:9000 -p 9001:9001 \
      --name minio \
      -e "MINIO_ROOT_USER=admin" \
      -e "MINIO_ROOT_PASSWORD=password" \
      minio/minio server /data --console-address ":9001"
    ```

2.  **Start Gateway with Environment Variables:**
    ```bash
    export MINIO_ENDPOINT="localhost:9000"
    export MINIO_ACCESS_KEY="admin"
    export MINIO_SECRET_KEY="password"
    
    go run cmd/gateway/*.go
    ```

*On startup, the Gateway will check the bucket for existing WASM binaries matching the code hash. If found, it skips compilation. This is useful for auto scaling or relevant tasks*

## ⚡ Quick Start

1.  Create a configuration file `configs/my_api.json`. Note the use of `inline_source` for custom logic.

    ```json
    {
      "api": "my_api",
      "mode": "ir",
      "version": "v2",
      "user_code": {
        "inline_source": "package main\n\n//export double\nfunc double(x uint64) uint64 { return x * 2 }\n\nfunc main() {}",
        "timeout_ms": 100
      },
      "routes": [
        {
          "method": "GET",
          "path": "/calc/{val}",
          "transform": {
            "extract": { "num": { "from": "path.val", "as": "x", "type": "int" } }
          },
          "response": {
            "status": 200,
            "body": { "result": "{{double(x)}}" }
          }
        }
      ]
    }
    ```

2.  **Test the Endpoint**:
    Routing is based on the **Host header** matching the `api` field.

    ```bash
    curl -H "Host:my_api.localhost" http://localhost:8080/calc/5
    ```

    **Response:**
    ```json
    { "result": "10" }
    ```

## 🏗️ Architecture

```mermaid
graph TD
    User[Client Request] --> Gateway
    Watcher[File Watcher] --> Builder
    
    subgraph Compiler Service
        Builder -->|Go Source| Compiler
        Compiler -->|Check Hash| L1_Memory[L1 Memory Cache]
        Compiler -->|Check Hash| L2_S3[L2 MinIO/S3]
        Compiler -->|Compile (TinyGo)| WASM_Bin
        WASM_Bin --> L2_S3
    end
    
    Compiler -->|WASM Bytes| Builder
    Builder --> Registry[Handler Registry]
    
    Gateway -->|Host Matching| Registry
    Registry -->|Get Handler| V2Engine
    
    subgraph V2Engine [Parallel Request Lifecycle]
        Validation[JSON Schema / Headers] --> Transformation
        Transformation -->|Extract| Context
        Transformation -->|ErrGroup: Parallel HTTP| Upstream[Upstream APIs]
        Transformation -->|WASM (Pooled)| Runtime[Wazero Runtime]
        Upstream --> Context
        Runtime --> Context
        Context --> ResponseBuilder
        ResponseBuilder -->|Latency Simulation| User
    end
```

## 📂 Project Structure

```text
.
├── cmd/gateway/           # Entry point, Hot-reload, GC, Build orchestration
├── configs/               # API JSON Configurations
├── internal/
│   ├── adapters/          # Config Parsers (V1/V2)
│   ├── config/            # Loader & Version Detection
│   ├── engine/            # Core Runtime (Router, HTTP, Validation, SafeContext)
│   ├── ir/                # Intermediate Representation Definitions
│   ├── services/
│   │   ├── compiler/      # Dynamic TinyGo Compiler & Hash Logic
│   │   ├── storage/       # S3/MinIO Client
│   │   └── wasm/          # Wazero Pool Manager & Memory Bridge
│   └── utils/             # Helpers
```

## 📚 Documentation

For a detailed guide on the JSON structure, including **JSON Schema Validation**, **WASM String/JSON processing**, and **Delay configuration**, please see the **[Configuration Guide](CONFIG_GUIDE.md)**.

## 🧪 Testing

Run unit tests and the End-to-End (E2E) test suite which compiles a real binary and hits it with requests, Use test/ branches for this:

```bash
go test ./... -v
```

***
