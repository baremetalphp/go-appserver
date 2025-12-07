package server

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type Server struct {
	fastPool *WorkerPool
	slowPool *WorkerPool
}

func NewServer(fastCount, slowCount int) (*Server, error) {
	fp, err := NewPool(fastCount)
	if err != nil {
		return nil, err
	}

	sp, err := NewPool(slowCount)
	if err != nil {
		return nil, err
	}

	return &Server{
		fastPool: fp,
		slowPool: sp,
	}, nil
}

// Classification logic -----------------------

func (s *Server) IsSlowRequest(r *RequestPayload) bool {
	// example heuristics

	//explicit slow routes (reports, exports)
	if strings.HasPrefix(r.Path, "/reports/") {
		return true
	}
	if strings.HasPrefix(r.Path, "/admin/analytics") {
		return true
	}

	// big uploads
	if len(r.Body) > 2_000_000 {
		return true
	}

	// PUT/DELETE often heavier
	if r.Method == "PUT" || r.Method == "DELETE" {
		return true
	}

	return false
}

// Dispatch -----------------------
func (s *Server) Dispatch(req *RequestPayload) (*ResponsePayload, error) {
	if s.IsSlowRequest(req) {
		return s.slowPool.Dispatch(req)
	}
	return s.fastPool.Dispatch(req)
}

// markAllWorkersDead forces both pools to recreate workers on next request
func (s *Server) markAllWorkersDead() {
	for _, w := range s.fastPool.workers {
		w.markDead()
	}
	for _, w := range s.slowPool.workers {
		w.markDead()
	}
}

// EnableHotReload watches PHP and routes directories in dev mode
// and marks all workers dead when code changes so they restart lazily
func (s *Server) EnableHotReload(projectRoot string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// directories to watch
	watchDirs := []string{
		filepath.Join(projectRoot, "php"),
		filepath.Join(projectRoot, "routes"),
	}

	for _, dir := range watchDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			if err := watcher.Add(dir); err != nil {
				log.Println("hot reload: failed to watch", dir, ":", err)
			} else {
				log.Println("hot reload: watching", dir)
			}
		}
	}

	go func() {
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
					log.Println("hot reload: detected change in", ev.Name, "- recycling workers...")
					s.markAllWorkersDead()
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("hot reload: watcher error:", err)
			}
		}
	}()

	return nil
}
