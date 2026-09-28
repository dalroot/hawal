package carrier

import (
	"fmt"
	"sort"
	"sync"
)

// Factory constructs an independent carrier instance.
type Factory func() (Carrier, error)

// Registry owns available carrier implementations. It is safe for concurrent
// reads and supports explicit registration during process startup.
type Registry struct {
	mu        sync.RWMutex
	factories map[Kind]Factory
}

func NewRegistry() *Registry {
	return &Registry{factories: make(map[Kind]Factory)}
}

func (r *Registry) Register(kind Kind, factory Factory) error {
	if kind == "" || factory == nil {
		return fmt.Errorf("carrier registry: kind and factory are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[kind]; exists {
		return fmt.Errorf("carrier registry: %q already registered", kind)
	}
	r.factories[kind] = factory
	return nil
}

func (r *Registry) Build(kind Kind) (Carrier, error) {
	r.mu.RLock()
	factory, exists := r.factories[kind]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("carrier registry: %q is not registered", kind)
	}
	instance, err := factory()
	if err != nil {
		return nil, fmt.Errorf("carrier registry: build %q: %w", kind, err)
	}
	if instance == nil || instance.Kind() != kind {
		return nil, fmt.Errorf("carrier registry: factory returned mismatched %q carrier", kind)
	}
	return instance, nil
}

func (r *Registry) Kinds() []Kind {
	r.mu.RLock()
	kinds := make([]Kind, 0, len(r.factories))
	for kind := range r.factories {
		kinds = append(kinds, kind)
	}
	r.mu.RUnlock()
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}
