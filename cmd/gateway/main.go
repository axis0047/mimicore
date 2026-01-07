package main

import (
	"log"
	"net/http"
	"os"

	"github.com/axis0047/mockingGOD/internal/engine"
	"github.com/axis0047/mockingGOD/internal/services/storage"
)

func main() {
	// 1. Init Storage (Optional: Only if ENV vars are present)
	if os.Getenv("MINIO_ENDPOINT") != "" {
		err := storage.Init(storage.S3Config{
			Endpoint:        os.Getenv("MINIO_ENDPOINT"),
			AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
			BucketName:      "mockinggod-wasm",
			UseSSL:          false, // Set true for prod
		})
		if err != nil {
			log.Fatalf("Failed to connect to MinIO: %v", err)
		}

		// Run GC on startup to clean old mess
		go RunGarbageCollection("configs")
	}

	registry := engine.NewRegistry()

	handlers, err := buildHandlers("configs")
	if err != nil {
		log.Fatal(err)
	}
	registry.ReplaceAll(handlers)

	go watchConfigs("configs", registry)

	gateway := &engine.Gateway{Registry: registry}

	log.Println("Gateway listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", gateway))
}
