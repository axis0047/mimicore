package engine

import (
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
	m := r.value.Load().(map[string]APIHandler)
	h, ok := m[api]
	return h, ok
}

func (r *Registry) Swap(next map[string]APIHandler) {
	r.value.Store(next)
}

func (r *Registry) ReplaceAll(next map[string]APIHandler) {
	r.value.Store(next)
}
