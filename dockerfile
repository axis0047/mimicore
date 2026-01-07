# Stage 1: Build the Gateway
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o gateway ./cmd/gateway

# Stage 2: Runtime Environment
FROM debian:bookworm-slim

# 1. Install dependencies for TinyGo
RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*

# 2. Install TinyGo
# FIX: Updated version to 0.33.0 (or latest) to ensure compatibility with Go 1.24 code
ENV TINYGO_VERSION=0.33.0
RUN wget https://github.com/tinygo-org/tinygo/releases/download/v${TINYGO_VERSION}/tinygo_${TINYGO_VERSION}_amd64.deb \
    && dpkg -i tinygo_${TINYGO_VERSION}_amd64.deb \
    && rm tinygo_${TINYGO_VERSION}_amd64.deb

# 3. Setup App
WORKDIR /app
COPY --from=builder /app/gateway .

# 4. Create configs directory
RUN mkdir -p configs

# 5. Expose Port
EXPOSE 8080

# 6. Run
CMD ["./gateway", "configs"]
