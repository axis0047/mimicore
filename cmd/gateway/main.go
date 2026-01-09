package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/axis0047/mockingGOD/internal/engine"
	"github.com/axis0047/mockingGOD/internal/middleware"
	"github.com/axis0047/mockingGOD/internal/services/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 0. Parse Configuration Directory
	configDir := "configs"
	if len(os.Args) > 1 {
		configDir = os.Args[1]
	}

	// 1. Start Metrics Server (Background)
	// We run this on a separate port (9090) so internal metrics aren't exposed
	// to the public API consumers on port 8080.
	go func() {
		log.Println("📊 Metrics exposed on :9090/metrics")
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 2. Initialize S3/MinIO Storage with Retry Logic
	// Docker containers often start at different speeds. We wait for MinIO.
	if os.Getenv("MINIO_ENDPOINT") != "" {
		var err error
		maxRetries := 30

		log.Println("Connecting to MinIO...")

		for i := 0; i < maxRetries; i++ {
			err = storage.Init(storage.S3Config{
				Endpoint:        os.Getenv("MINIO_ENDPOINT"),
				AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
				SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
				BucketName:      "mockinggod-wasm",
				UseSSL:          false, // Set to true in production if using HTTPS
			})

			if err == nil {
				log.Println("✅ S3 Storage Initialized")
				break
			}

			// Log warning but don't crash yet
			log.Printf("MinIO not ready yet (%v). Retrying in 1s... [%d/%d]", err, i+1, maxRetries)
			time.Sleep(1 * time.Second)
		}

		if err != nil {
			// Only crash if we failed 30 times (30 seconds)
			log.Fatalf("❌ Failed to connect to MinIO after %d retries: %v", maxRetries, err)
		}

		// Run Garbage Collection on startup to clean up orphaned WASM blobs
		go RunGarbageCollection(configDir)
	}

	// 3. Initialize Core Engine
	registry := engine.NewRegistry()

	// Initial Load
	handlers, err := buildHandlers(configDir)
	if err != nil {
		log.Printf("Startup warning: %v", err)
	}
	registry.ReplaceAll(handlers)

	// Start Hot-Reload Watcher
	go watchConfigs(configDir, registry)

	// 4. Setup Gateway & Middleware
	gateway := &engine.Gateway{Registry: registry}

	// Wrap the gateway with Metrics Middleware
	// This ensures every request is counted and timed
	handler := middleware.MetricsMiddleware(gateway)

	log.Println("Gateway listening on :8080")
	log.Printf("Reading configs from: %s", configDir)

	// 5. Start Traffic Server
	log.Fatal(http.ListenAndServe(":8080", handler))
}
