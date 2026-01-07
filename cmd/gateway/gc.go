package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	v2 "github.com/axis0047/mockingGOD/internal/adapters/v2"
	"github.com/axis0047/mockingGOD/internal/services/storage"
)

// RunGarbageCollection deletes S3 blobs that are not referenced by any config
func RunGarbageCollection(configDir string) {
	if storage.GlobalStorage == nil {
		return
	}
	log.Println("🧹 Starting WASM Garbage Collection...")

	// 1. Collect Valid Hashes
	validHashes := make(map[string]bool)

	files, _ := filepath.Glob(filepath.Join(configDir, "*.json"))
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		var cfg v2.APIFileConfig
		if err := json.Unmarshal(raw, &cfg); err == nil && cfg.UserCode != nil {
			if cfg.UserCode.InlineSource != "" {
				hasher := sha256.New()
				hasher.Write([]byte(cfg.UserCode.InlineSource))
				hash := hex.EncodeToString(hasher.Sum(nil))

				// Expected S3 key
				key := "wasm-cache/" + hash + ".wasm"
				validHashes[key] = true
			}
		}
	}

	// 2. List S3 Objects
	existingObjects, err := storage.GlobalStorage.List()
	if err != nil {
		log.Printf("GC Failed to list objects: %v", err)
		return
	}

	// 3. Identify Orphans
	var toDelete []string
	for _, key := range existingObjects {
		if !validHashes[key] {
			log.Printf("🗑️  Marked for deletion: %s", key)
			toDelete = append(toDelete, key)
		}
	}

	// 4. Delete
	if len(toDelete) > 0 {
		if err := storage.GlobalStorage.Delete(toDelete); err != nil {
			log.Printf("GC Error deleting: %v", err)
		} else {
			log.Printf("✅ GC Removed %d orphaned binaries", len(toDelete))
		}
	} else {
		log.Println("✅ Storage is clean. No orphans found.")
	}
}
