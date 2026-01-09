# --------------------------------------------------------
# STAGE 1: Build the Gateway (Using Go 1.24)
# This satisfies your go.mod requirement.
# --------------------------------------------------------
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build the binary
RUN go build -o gateway ./cmd/gateway

# --------------------------------------------------------
# STAGE 2: Runtime Environment (Using Go 1.23)
# We use Go 1.23 here specifically because TinyGo needs
# the Go 1.23 source code (/usr/local/go) to compile WASM.
# --------------------------------------------------------
FROM golang:1.23-bookworm

# 1. Install dependencies
RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*

# 2. Install TinyGo 0.33.0
# This version works perfectly with the Go 1.23 environment we provided above
ENV TINYGO_VERSION=0.33.0
RUN wget https://github.com/tinygo-org/tinygo/releases/download/v${TINYGO_VERSION}/tinygo_${TINYGO_VERSION}_amd64.deb \
    && dpkg -i tinygo_${TINYGO_VERSION}_amd64.deb \
    && rm tinygo_${TINYGO_VERSION}_amd64.deb

# 3. Setup App
WORKDIR /app

# Copy the binary compiled in Stage 1
COPY --from=builder /app/gateway .

# 4. Create configs directory
RUN mkdir -p configs

# 5. Expose Ports
EXPOSE 8080 9090

# 6. Run
CMD ["./gateway", "configs"]
