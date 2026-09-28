package session

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

type PathState uint8

const (
	PathStandby PathState = iota + 1
	PathPrimary
	PathDegraded
)

var (
	ErrPathNotFound      = errors.New("session: path not found")
	ErrGenerationChanged = errors.New("session: path generation changed")
	ErrDetachPrimary     = errors.New("session: cannot detach primary path")
)

type PathSnapshot struct {
	ID          string
	Kind        carrier.Kind
	State       PathState
	AttachedAt  time.Time
	LastChanged time.Time
}

type pathEntry struct {
	link carrier.Link
	PathSnapshot
}

// PathSet tracks multiple live carriers without treating attachment of a new
// path as permission to close the current primary path.
type PathSet struct {
	mu         sync.RWMutex
	paths      map[string]*pathEntry
	primary    string
	generation uint64
	maxPaths   int
}

func NewPathSet(maxPaths int) (*PathSet, error) {
	if maxPaths < 1 || maxPaths > 64 {
		return nil, errors.New("session: max paths outside safe range")
	}
	return &PathSet{paths: make(map[string]*pathEntry), maxPaths: maxPaths}, nil
}

// Attach is idempotent for the same Link ID. The first path becomes primary;
// later paths remain standby until an explicit promotion succeeds.
func (p *PathSet) Attach(link carrier.Link) (bool, error) {
	if link == nil || link.ID() == "" {
		return false, errors.New("session: link and link ID are required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.paths[link.ID()]; ok {
		return false, nil
	}
	if len(p.paths) == p.maxPaths {
		return false, errors.New("session: path limit reached")
	}
	now := time.Now().UTC()
	state := PathStandby
	if p.primary == "" {
		state = PathPrimary
		p.primary = link.ID()
	}
	p.paths[link.ID()] = &pathEntry{link: link, PathSnapshot: PathSnapshot{ID: link.ID(), Kind: link.Kind(), State: state, AttachedAt: now, LastChanged: now}}
	p.generation++
	return true, nil
}

// Promote uses optimistic generation checking. Repeating a completed promotion
// is harmless, while a competing topology change forces the caller to re-read.
func (p *PathSet) Promote(id string, expectedGeneration uint64) (uint64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.primary == id {
		return p.generation, nil
	}
	if p.generation != expectedGeneration {
		return p.generation, ErrGenerationChanged
	}
	next, ok := p.paths[id]
	if !ok {
		return p.generation, ErrPathNotFound
	}
	now := time.Now().UTC()
	if current := p.paths[p.primary]; current != nil {
		current.State = PathStandby
		current.LastChanged = now
	}
	next.State = PathPrimary
	next.LastChanged = now
	p.primary = id
	p.generation++
	return p.generation, nil
}

func (p *PathSet) MarkDegraded(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.paths[id]
	if !ok {
		return ErrPathNotFound
	}
	entry.State = PathDegraded
	entry.LastChanged = time.Now().UTC()
	p.generation++
	return nil
}

// Detach removes a standby/degraded path and transfers Link ownership to the
// caller. A primary must first be explicitly migrated, preventing accidental
// session loss during cleanup.
func (p *PathSet) Detach(id string) (carrier.Link, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.primary == id {
		return nil, ErrDetachPrimary
	}
	entry, ok := p.paths[id]
	if !ok {
		return nil, ErrPathNotFound
	}
	delete(p.paths, id)
	p.generation++
	return entry.link, nil
}

func (p *PathSet) Snapshot() (primary string, generation uint64, paths []PathSnapshot) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	paths = make([]PathSnapshot, 0, len(p.paths))
	for _, entry := range p.paths {
		paths = append(paths, entry.PathSnapshot)
	}
	sort.Slice(paths, func(i, j int) bool { return paths[i].ID < paths[j].ID })
	return p.primary, p.generation, paths
}
