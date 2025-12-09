package server

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type WorkerState int

const (
	WorkerIdle WorkerState = iota
	WorkerBusy
	WorkerDraining
	WorkerDead
)

type Worker struct {
	cmd            *exec.Cmd
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	mu             sync.Mutex // protects cmd/stdin/stdout during request I/O
	baseDir        string
	dead           bool
	deadMu         sync.RWMutex // protects dead flag
	maxRequests    int
	requestTimeout time.Duration
	requestCount   uint64

	stateMu  sync.RWMutex // protects state + inFlight
	state    WorkerState
	inFlight int
}

// NewWorker walks up from the current directory to find go.mod,
// assumes php/worker.php relative to that, and starts a PHP worker.
func NewWorker(maxRequests int, requestTimeout time.Duration) (*Worker, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	baseDir := wd
	for {
		if _, err := os.Stat(filepath.Join(baseDir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(baseDir)
		if parent == baseDir {
			break
		}
		baseDir = parent
	}

	workerPath := filepath.Join(baseDir, "php", "worker.php")

	cmd := exec.Command("php", workerPath)
	cmd.Dir = baseDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}

	cmd.Stderr = log.Writer()

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}

	return &Worker{
		cmd:            cmd,
		stdin:          stdin,
		stdout:         stdout,
		baseDir:        baseDir,
		dead:           false,
		maxRequests:    maxRequests,
		requestTimeout: requestTimeout,
		state:          WorkerIdle,
	}, nil
}

func (w *Worker) isDead() bool {
	w.deadMu.RLock()
	dead := w.dead
	w.deadMu.RUnlock()
	return dead
}

func (w *Worker) markDead() {
	w.deadMu.Lock()
	w.dead = true
	w.deadMu.Unlock()

	w.stateMu.Lock()
	w.state = WorkerDead
	w.stateMu.Unlock()
}

func (w *Worker) setState(state WorkerState) {
	w.stateMu.Lock()
	w.state = state
	w.stateMu.Unlock()
}

func (w *Worker) getState() WorkerState {
	w.stateMu.RLock()
	s := w.state
	w.stateMu.RUnlock()
	return s
}

func (w *Worker) incrInFlight() {
	w.stateMu.Lock()
	w.inFlight++
	w.stateMu.Unlock()
}

func (w *Worker) decrInFlight() {
	w.stateMu.Lock()
	if w.inFlight > 0 {
		w.inFlight--
	}
	w.stateMu.Unlock()
}

func (w *Worker) getInFlight() int {
	w.stateMu.RLock()
	n := w.inFlight
	w.stateMu.RUnlock()
	return n
}

func (w *Worker) startDraining() {
	w.stateMu.Lock()
	if w.state != WorkerDead {
		w.state = WorkerDraining
	}
	w.stateMu.Unlock()
}

func (w *Worker) isDraining() bool {
	w.stateMu.RLock()
	draining := w.state == WorkerDraining
	w.stateMu.RUnlock()
	return draining
}

func (w *Worker) restart() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stdin != nil {
		_ = w.stdin.Close()
	}
	if w.stdout != nil {
		_ = w.stdout.Close()
	}
	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
		_, _ = w.cmd.Process.Wait()
	}

	workerPath := filepath.Join(w.baseDir, "php", "worker.php")
	cmd := exec.Command("php", workerPath)
	cmd.Dir = w.baseDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}

	cmd.Stderr = log.Writer()

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return err
	}

	w.cmd = cmd
	w.stdin = stdin
	w.stdout = stdout

	w.deadMu.Lock()
	w.dead = false
	w.deadMu.Unlock()

	w.stateMu.Lock()
	w.state = WorkerIdle
	w.inFlight = 0
	w.stateMu.Unlock()

	atomic.StoreUint64(&w.requestCount, 0)

	log.Println("Restarted PHP worker in", w.baseDir)

	return nil
}

func (w *Worker) Handle(payload *RequestPayload) (*ResponsePayload, error) {
	if w.isDead() {
		return nil, ErrWorkerDead
	}

	// don't send new work to draining workers
	if w.isDraining() {
		return nil, ErrWorkerDraining
	}

	w.incrInFlight()
	w.setState(WorkerBusy)
	defer func() {
		w.decrInFlight()
		if w.getInFlight() == 0 && w.isDraining() {
			// safe to recycle
			w.markDead()
		} else if !w.isDead() {
			w.setState(WorkerIdle)
		}
	}()

	for attempt := 0; attempt < 2; attempt++ {
		if w.isDead() {
			if err := w.restart(); err != nil {
				return nil, err
			}
		}

		resp, err := w.handleRequest(payload)
		if err != nil {
			if isBrokenPipe(err) {
				w.markDead()
				continue
			}
			return nil, err
		}

		// increment request count and recycle if exceeding maxRequests
		n := atomic.AddUint64(&w.requestCount, 1)
		if w.maxRequests > 0 && int(n) >= w.maxRequests {
			w.markDead()
		}

		return resp, nil
	}

	return nil, io.ErrUnexpectedEOF
}

func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return err == io.EOF ||
		err == io.ErrUnexpectedEOF ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "write |1:") ||
		strings.Contains(errStr, "read |0:")
}

func (w *Worker) handleRequest(payload *RequestPayload) (*ResponsePayload, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	length := uint32(len(jsonBytes))

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length)

	if _, err := w.stdin.Write(header); err != nil {
		return nil, err
	}
	if _, err := w.stdin.Write(jsonBytes); err != nil {
		return nil, err
	}

	type result struct {
		resp *ResponsePayload
		err  error
	}

	resCh := make(chan result, 1)

	go func() {
		// read length header
		hdr := make([]byte, 4)
		if _, err := io.ReadFull(w.stdout, hdr); err != nil {
			resCh <- result{nil, err}
			return
		}

		respLen := binary.BigEndian.Uint32(hdr)

		if respLen == 0 || respLen > 10*1024*1024 {
			resCh <- result{nil, io.ErrUnexpectedEOF}
			return
		}

		respJSON := make([]byte, respLen)
		if _, err := io.ReadFull(w.stdout, respJSON); err != nil {
			resCh <- result{nil, err}
			return
		}

		var resp ResponsePayload
		if err := json.Unmarshal(respJSON, &resp); err != nil {
			resCh <- result{nil, err}
			return
		}

		resCh <- result{&resp, nil}
	}()

	if w.requestTimeout > 0 {
		select {
		case res := <-resCh:
			return res.resp, res.err
		case <-time.After(w.requestTimeout):
			// Kill and mark dead on timeout
			w.markDead()
			if w.cmd != nil && w.cmd.Process != nil {
				_ = w.cmd.Process.Kill()
				_, _ = w.cmd.Process.Wait()
			}
			return nil, fmt.Errorf("worker request timeout after %s", w.requestTimeout)
		}
	}

	res := <-resCh
	return res.resp, res.err
}

// Stream sends the request and streams the response frames directly to the client.
func (w *Worker) Stream(req *RequestPayload, rw http.ResponseWriter) error {
	if w.isDead() || w.isDraining() {
		return ErrWorkerDead
	}

	w.incrInFlight()
	w.setState(WorkerBusy)
	defer func() {
		w.decrInFlight()
		if w.getInFlight() == 0 && w.isDraining() {
			w.markDead()
		} else if !w.isDead() {
			w.setState(WorkerIdle)
		}
	}()

	type result struct {
		err error
	}

	resCh := make(chan result, 1)

	go func() {
		resCh <- result{err: w.streamInternal(req, rw)}
	}()

	if w.requestTimeout > 0 {
		select {
		case res := <-resCh:
			return res.err
		case <-time.After(w.requestTimeout):
			// Kill and mark dead on timeout
			w.markDead()
			if w.cmd != nil && w.cmd.Process != nil {
				_ = w.cmd.Process.Kill()
				_, _ = w.cmd.Process.Wait()
			}
			return fmt.Errorf("worker stream timeout after %s", w.requestTimeout)
		}
	}

	res := <-resCh
	return res.err
}

// streamInternal performs the actual length-prefixed send/receive under lock.
func (w *Worker) streamInternal(req *RequestPayload, rw http.ResponseWriter) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isDead() {
		if err := w.restart(); err != nil {
			return err
		}
	}

	// 1) Encode and send the request as length-prefixed JSON
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}
	length := uint32(len(jsonBytes))

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length)

	if _, err := w.stdin.Write(header); err != nil {
		return err
	}
	if _, err := w.stdin.Write(jsonBytes); err != nil {
		return err
	}

	headersSent := false
	statusCode := http.StatusOK

	for {
		// 2) Read 4-byte frame length
		hdr := make([]byte, 4)
		if _, err := io.ReadFull(w.stdout, hdr); err != nil {
			w.markDead()
			return err
		}

		frameLen := binary.BigEndian.Uint32(hdr)

		if frameLen == 0 || frameLen > 10*1024*1024 {
			w.markDead()
			return io.ErrUnexpectedEOF
		}

		// 3) Read JSON frame
		frameJSON := make([]byte, frameLen)
		if _, err := io.ReadFull(w.stdout, frameJSON); err != nil {
			w.markDead()
			return err
		}

		var frame StreamFrame
		if err := json.Unmarshal(frameJSON, &frame); err != nil {
			w.markDead()
			return err
		}

		switch frame.Type {
		case "headers":
			if frame.Headers != nil {
				for k, vs := range frame.Headers {
					if len(vs) == 0 {
						continue
					}

					if strings.ToLower(k) == "set-cookie" {
						// can't join, must be dealt with separately
						for _, v := range vs {
							rw.Header().Add(k, v)
						}
					} else {
						// RFC-compliant: join
						rw.Header().Set(k, strings.Join(vs, ", "))
					}

				}
			}
			if frame.Status != 0 {
				statusCode = frame.Status
			}
			rw.WriteHeader(statusCode)
			headersSent = true

			if frame.Data != "" {
				if _, err := rw.Write([]byte(frame.Data)); err != nil {
					return err
				}
				if f, ok := rw.(http.Flusher); ok {
					f.Flush()
				}
			}

		case "chunk":
			if !headersSent {
				rw.WriteHeader(statusCode)
				headersSent = true
			}
			if frame.Data != "" {
				if _, err := rw.Write([]byte(frame.Data)); err != nil {
					return err
				}
				if f, ok := rw.(http.Flusher); ok {
					f.Flush()
				}
			}

		case "end":
			// Normal end of stream
			return nil

		case "error":
			return fmt.Errorf("stream error from worker: %s", frame.Error)

		default:
			return fmt.Errorf("unknown stream frame type: %q", frame.Type)
		}
	}
}
