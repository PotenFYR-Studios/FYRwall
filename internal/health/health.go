// Package health implements the startup state machine and aggregated
// health state (spec sections 9, 10.2, 44).
package health

import (
	"sync"
	"time"
)

// State is the overall application health.
type State string

const (
	StateStarting State = "STARTING"
	StateHealthy  State = "HEALTHY"
	StateDegraded State = "DEGRADED"
	StateBlocked  State = "BLOCKED"
	StateStopping State = "STOPPING"
)

// ComponentState is one component's contribution to overall health.
type ComponentState struct {
	Name    string    `json:"name"`
	Status  State     `json:"status"`
	Detail  string    `json:"detail,omitempty"`
	Checked time.Time `json:"checked"`
}

// Tracker aggregates component states into overall health (spec section 9:
// BLOCKED if any mandatory component fails, DEGRADED on warnings).
type Tracker struct {
	mu         sync.RWMutex
	components map[string]ComponentState
	startedAt  time.Time
}

// NewTracker starts in STARTING state.
func NewTracker() *Tracker {
	return &Tracker{
		components: map[string]ComponentState{},
		startedAt:  time.Now().UTC(),
	}
}

// Set records a component state.
func (t *Tracker) Set(name string, st State, detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.components[name] = ComponentState{
		Name: name, Status: st, Detail: detail, Checked: time.Now().UTC(),
	}
}

// Overall computes the aggregate state.
func (t *Tracker) Overall() State {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.components) == 0 {
		return StateStarting
	}
	blocked, degraded := false, false
	for _, c := range t.components {
		switch c.Status {
		case StateBlocked:
			blocked = true
		case StateDegraded:
			degraded = true
		}
	}
	switch {
	case blocked:
		return StateBlocked
	case degraded:
		return StateDegraded
	default:
		return StateHealthy
	}
}

// Snapshot returns all component states plus the overall.
func (t *Tracker) Snapshot() (State, []ComponentState) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]ComponentState, 0, len(t.components))
	for _, c := range t.components {
		out = append(out, c)
	}
	return t.Overall(), out
}

// Uptime returns time since tracker creation.
func (t *Tracker) Uptime() time.Duration { return time.Since(t.startedAt) }
