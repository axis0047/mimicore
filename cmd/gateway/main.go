package main

import (
	"log"
	"net/http"
	"os"
	"time" // <--- Add this import

	"github.com/axis0047/mockingGOD/internal/engine"
	"github.com/axis0047/mockingGOD/internal/middleware"
	"github.com/axis0047/mockingGOD/internal/services/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configDir := "configs"
	if len(os.Args) > 1 {
		configDir = os.Args[1]
	}

	// 1. Start Metrics (Background)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		// log.Fatal(http.ListenAndServe(":9090", nil)) // Usually separate port, handled below or separate go routine
		// For simplicity in this structure:
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 2. Init Storage with Retry Logic
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
				UseSSL:          false,
			})

			if err == nil {
				break // Success!
			}

			// Log warning but don't crash
			log.Printf("MinIO not ready yet (%v). Retrying in 1s... [%d/%d]", err, i+1, maxRetries)
			time.Sleep(1 * time.Second)
		}

		if err != nil {
			// Only crash if we failed 30 times (30 seconds)
			log.Fatalf("❌ Failed to connect to MinIO after %d retries: %v", maxRetries, err)
		}

		// Run GC on startup
		go RunGarbageCollection(configDir)
	}

	registry := engine.NewRegistry()

	handlers, err := buildHandlers(configDir)
	if err != nil {
		log.Printf("Startup warning: %v", err)
	}
	registry.ReplaceAll(handlers)

	go watchConfigs(configDir, registry)

	gateway := &engine.Gateway{Registry: registry}

	// Wrap with Middleware
	handler := middleware.MetricsMiddleware(gateway)

	log.Println("Gateway listening on :8080")
	log.Printf("Reading configs from: %s", configDir)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
