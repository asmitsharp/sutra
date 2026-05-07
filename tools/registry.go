package tools

import (
	"fmt"
	"sync"
)

type Registry struct {
	m sync.Map
}

func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a tool to the registry
func (r *Registry) Register(t Tool) error {
	if _, loaded := r.m.LoadOrStore(t.Name(), t); loaded {
		return fmt.Errorf("tool %s already registered", t.Name())
	}
	return nil
}

// MustRegister panics if the tool is already registered
func (r *Registry) MustRegister(t Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	val, ok := r.m.Load(name)
	if !ok {
		return nil, false
	}
	return val.(Tool), true
}

// List returns a slice of all registered tools
func (r *Registry) List() []Tool {
	var tools []Tool
	r.m.Range(func(_ any, value any) bool {
		tools = append(tools, value.(Tool))
		return true
	})
	return tools
}
