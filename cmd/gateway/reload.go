package main

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/axis0047/mockingGOD/internal/engine"
	"github.com/fsnotify/fsnotify"
)

func watchConfigs(dir string, registry *engine.Registry) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	if err := watcher.Add(dir); err != nil {
		log.Fatal(err)
	}

	log.Println("Watcher started on", dir)

	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only care about JSON files
			if filepath.Ext(ev.Name) != ".json" {
				continue
			}

			// Debouncing logic can be complex in granular updates.
			// For simplicity, we process immediately, but a production system might
			// use a map of timers per file.

			// Handle REMOVE / RENAME (Old file gone)
			if ev.Op&fsnotify.Remove == fsnotify.Remove || ev.Op&fsnotify.Rename == fsnotify.Rename {
				apiName := strings.TrimSuffix(filepath.Base(ev.Name), ".json")
				log.Printf("Config removed/renamed: %s", apiName)
				registry.Remove(apiName)
			}

			// Handle CREATE / WRITE (New/Modified file)
			if ev.Op&fsnotify.Create == fsnotify.Create || ev.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("Config changed: %s", ev.Name)

				// Rebuild ONLY this file
				apiName, handler, err := buildFromPath(ev.Name)
				if err != nil {
					log.Printf("Failed to reload %s: %v", ev.Name, err)
					continue
				}

				// Granular Update
				registry.Register(apiName, handler)
				log.Printf("✓ Reloaded API: %s", apiName)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Watcher error:", err)
		}
	}
}
