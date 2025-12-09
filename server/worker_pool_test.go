package server

import (
	"testing"
	"time"
)

// NOTE: these tests operate only on the Worker state flags and pool logic.
// we do not need real php worker processes; zero-value *Worker is fine
// as long as we only use state helpers (markDead, startDraining, etc.)

func TestNewPoolCreatesCorrectNumberOfWorkers(t *testing.T) {
	poolSize := 3
	pool, err := NewPool(poolSize, 10, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("NewPool returned error: %v", err)
	}
	if pool == nil {
		t.Fatal("NewPool returned nil pool")
	}
	if got := len(pool.workers); got != poolSize {
		t.Fatalf("expected %d workers, got %d", poolSize, got)
	}
}
func TestNextWorkerSkipsDeadAndDraining(t *testing.T) {
	// Three workers: w1 (dead), w2 (draining), w3 (healthy)
	w1 := &Worker{}
	w2 := &Worker{}
	w3 := &Worker{}

	w1.markDead()
	w2.startDraining()

	pool := &WorkerPool{
		workers: []*Worker{w1, w2, w3},
	}

	// First call should skip w1 (dead) and w2 (draining) and return w3.
	w := pool.NextWorker()
	if w != w3 {
		t.Fatalf("expected NextWorker to return w3, got %#v", w)
	}

	// Subsequent calls should still return w3, since the others are unusable.
	for i := 0; i < 3; i++ {
		w = pool.NextWorker()
		if w != w3 {
			t.Fatalf("expected NextWorker to keep returning w3, got %#v on iteration %d", w, i)
		}
	}
}

func TestDrainAllMarksWorkersAsDraining(t *testing.T) {
	w1 := &Worker{}
	w2 := &Worker{}
	w3 := &Worker{}
	pool := &WorkerPool{
		workers: []*Worker{w1, w2, w3},
	}

	pool.DrainAll()

	for i, w := range []*Worker{w1, w2, w3} {
		if w.isDead() {
			t.Fatalf("worker %d should be draining, not dead", i+1)
		}
		if !w.isDraining() {
			t.Fatalf("worker %d should be marked draining", i+1)
		}
	}
}

func TestScaleToShrinkMarksExtrasDrainingAndTruncatesSlice(t *testing.T) {
	w1 := &Worker{}
	w2 := &Worker{}
	w3 := &Worker{}
	pool := &WorkerPool{
		workers: []*Worker{w1, w2, w3},
	}

	// Shrink from 3 -> 1
	if err := pool.ScaleTo(1, nil); err != nil {
		t.Fatalf("ScaleTo(1) returned error: %v", err)
	}

	if got := len(pool.workers); got != 1 {
		t.Fatalf("expected pool size 1 after shrink, got %d", got)
	}
	if pool.workers[0] != w1 {
		t.Fatalf("expected remaining worker to be w1 after shrink")
	}

	// w2 and w3 should have been marked draining
	if !w2.isDraining() || w2.isDead() {
		t.Fatalf("w2 should be draining (not dead) after shrink")
	}
	if !w3.isDraining() || w3.isDead() {
		t.Fatalf("w3 should be draining (not dead) after shrink")
	}
}

func TestScaleToGrowUsesFactory(t *testing.T) {
	// Start with one worker
	w1 := &Worker{}
	pool := &WorkerPool{
		workers: []*Worker{w1},
	}

	var created int
	factory := func() (*Worker, error) {
		created++
		return &Worker{}, nil
	}

	// Grow from 1 -> 3
	if err := pool.ScaleTo(3, factory); err != nil {
		t.Fatalf("ScaleTo(3) returned error: %v", err)
	}

	if got := len(pool.workers); got != 3 {
		t.Fatalf("expected pool size 3 after grow, got %d", got)
	}
	if created != 2 {
		t.Fatalf("expected factory to be called 2 times, got %d", created)
	}
	if pool.workers[0] != w1 {
		t.Fatalf("expected original worker to remain at index 0")
	}
}

func TestStatsCountsDeadWorkers(t *testing.T) {
	w1 := &Worker{}
	w2 := &Worker{}
	w3 := &Worker{}

	w2.markDead()

	pool := &WorkerPool{
		workers: []*Worker{w1, w2, w3},
	}

	stats := pool.Stats()
	if stats.Workers != 3 {
		t.Fatalf("expected Workers=3, got %d", stats.Workers)
	}
	if stats.DeadWorkers != 1 {
		t.Fatalf("expected DeadWorkers=1, got %d", stats.DeadWorkers)
	}
}
