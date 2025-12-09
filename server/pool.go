package server

import (
	"errors"
	"sync"
	"time"
)

var ErrNoWorkers = errors.New("no workers available")

type WorkerPool struct {
	workers []*Worker
	mu      sync.Mutex
	next    int
}

// NewPool creates a pool with count workers, each configured
// with maxRequests and requestTimeout.
func NewPool(count int, maxRequests int, requestTimeout time.Duration) (*WorkerPool, error) {
	workers := make([]*Worker, 0, count)

	for i := 0; i < count; i++ {
		w, err := NewWorker(maxRequests, requestTimeout)
		if err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}

	return &WorkerPool{
		workers: workers,
	}, nil
}

func (p *WorkerPool) Dispatch(req *RequestPayload) (*ResponsePayload, error) {
	w := p.NextWorker()
	if w == nil {
		return nil, ErrNoWorkers
	}

	return w.Handle(req)
}
func (p *WorkerPool) Stats() PoolStats {
	stats := PoolStats{}
	if p == nil {
		return stats
	}

	stats.Workers = len(p.workers)
	for _, w := range p.workers {
		if w != nil && w.isDead() {
			stats.DeadWorkers++
		}
	}

	return stats
}

func (p *WorkerPool) NextWorker() *Worker {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.workers)
	if n == 0 {
		return nil
	}

	for i := 0; i < n; i++ {
		w := p.workers[p.next]
		p.next = (p.next + 1) % n
		if w != nil && !w.isDead() && !w.isDraining() {
			return w
		}
	}
	return nil
}

func (p *WorkerPool) DrainAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, w := range p.workers {
		if w != nil && !w.isDead() {
			w.startDraining()
		}
	}
}

// ScaleTo lets you grow/shrink the pool
func (p *WorkerPool) ScaleTo(newSize int, factory func() (*Worker, error)) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cur := len(p.workers)
	switch {
	case newSize == cur:
		return nil
	case newSize < cur:
		// mark extras as draining so they shut down after in-flight work
		for i := newSize; i < cur; i++ {
			if p.workers[i] != nil {
				p.workers[i].startDraining()
			}
		}
		p.workers = p.workers[:newSize]
		if p.next >= newSize {
			p.next = 0
		}
		return nil
	default: // grow
		for i := cur; i < newSize; i++ {
			w, err := factory()
			if err != nil {
				return err
			}
			p.workers = append(p.workers, w)
		}
		return nil
	}
}
