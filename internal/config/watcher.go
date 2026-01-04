package config

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

func Watch(path string, manager *Manager) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(path); err != nil {
		return err
	}

	go func() {
		var timer *time.Timer

		for {
			select {
			case ev := <-watcher.Events:
				if ev.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					if timer != nil {
						timer.Stop()
					}

					timer = time.AfterFunc(500*time.Millisecond, func() {
						manager.Reload()
					})
				}

			case err := <-watcher.Errors:
				log.Println("watcher error:", err)
			}
		}
	}()

	return nil
}
