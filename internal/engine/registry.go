package engine

import (
	"fmt"
	"sync/atomic"
)

type APIHandler interface {
	Serve(*Context)
}

type Registry struct {
	value atomic.Value // map[string]APIHandler
}

func NewRegistry() *Registry {
	r := &Registry{}
	r.value.Store(map[string]APIHandler{})
	return r
}

func (r *Registry) Get(api string) (APIHandler, bool) {
	loaded := r.value.Load()
	fmt.Printf("Loaded value: %+v, Type: %T\n", loaded, loaded)

	m := loaded.(map[string]APIHandler)
	fmt.Printf("Map contents: %+v\n", m)
	fmt.Printf("Looking for key: %q\n", api)

	h, ok := m[api]
	fmt.Printf("get function - %v - %v\n", h, ok)
	return h, ok
}

func (r *Registry) Swap(next map[string]APIHandler) {
	r.value.Store(next)
}

func (r *Registry) ReplaceAll(next map[string]APIHandler) {
	r.value.Store(next)
}
