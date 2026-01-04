package config

import (
	"log"

	"github.com/axis0047/mockingGOD/internal/engine"
	"github.com/axis0047/mockingGOD/internal/ir"
)

type Adapter interface {
	Compile() ([]ir.Route, error)
}

type Manager struct {
	Adapter Adapter
	Gateway *engine.Gateway
}

func (m *Manager) Reload() {
	routes, err := m.Adapter.Compile()
	if err != nil {
		log.Println("config reload failed:", err)
		return
	}

	router := &engine.Router{Routes: routes}
	m.Gateway.SwapRouter(router)

	log.Println("config reloaded successfully")
}
