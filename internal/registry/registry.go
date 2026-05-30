package registry

import (
	"fmt"
	"sync"

	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// Registry holds pluggable components keyed by name.
type Registry struct {
	mu sync.RWMutex

	segments map[string]plugin.SegmentProvider
	sources  map[string]plugin.ShoppingSource
	emails   map[string]plugin.EmailSender
	rewards  map[string]plugin.RewardProvider
}

func New() *Registry {
	return &Registry{
		segments: make(map[string]plugin.SegmentProvider),
		sources:  make(map[string]plugin.ShoppingSource),
		emails:   make(map[string]plugin.EmailSender),
		rewards:  make(map[string]plugin.RewardProvider),
	}
}

func (r *Registry) RegisterSegment(p plugin.SegmentProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.segments[p.Name()] = p
}

func (r *Registry) RegisterSource(p plugin.ShoppingSource) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[p.Name()] = p
}

func (r *Registry) RegisterEmail(p plugin.EmailSender) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.emails[p.Name()] = p
}

func (r *Registry) RegisterReward(p plugin.RewardProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rewards[p.Name()] = p
}

func (r *Registry) Segment(name string) (plugin.SegmentProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.segments[name]
	if !ok {
		return nil, fmt.Errorf("segment provider %q not registered", name)
	}
	return p, nil
}

func (r *Registry) Source(name string) (plugin.ShoppingSource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.sources[name]
	if !ok {
		return nil, fmt.Errorf("shopping source %q not registered", name)
	}
	return p, nil
}

func (r *Registry) Email(name string) (plugin.EmailSender, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.emails[name]
	if !ok {
		return nil, fmt.Errorf("email sender %q not registered", name)
	}
	return p, nil
}

func (r *Registry) Reward(name string) (plugin.RewardProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.rewards[name]
	if !ok {
		return nil, fmt.Errorf("reward provider %q not registered", name)
	}
	return p, nil
}

func (r *Registry) ListRewards() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.rewards))
	for name := range r.rewards {
		out = append(out, name)
	}
	return out
}
