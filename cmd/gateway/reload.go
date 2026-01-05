package main

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/axis0047/mockingGOD/internal/engine"
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

	var debounce *time.Timer

	reload := func() {
		handlers, err := buildHandlers(dir)
		if err != nil {
			log.Println("reload failed:", err)
			return
		}
		registry.ReplaceAll(handlers)
		log.Println("configs reloaded")
	}

	for {
		select {
		case ev := <-watcher.Events:
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(300*time.Millisecond, reload)
			}

		case err := <-watcher.Errors:
			log.Println("watcher error:", err)
		}
	}
}
