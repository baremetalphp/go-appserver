package server

import (
	"encoding/json"
	"log"
	"sync"
)

type sseEvent struct {
	Channel string
	Event   string
	Data    []byte
}

type sseClient struct {
	ch   chan sseEvent
	done chan struct{}
}

// Ch returns the event channel for the client
func (c *sseClient) Ch() <-chan sseEvent {
	return c.ch
}

// Done returns the done channel for the client
func (c *sseClient) Done() <-chan struct{} {
	return c.done
}

type SSEHub struct {
	mu       sync.RWMutex
	clients  map[string]map[*sseClient]struct{} // channel -> set of clients
	incoming chan sseEvent
}

// NewSSEHub creates a hub and starts its fanout goroutine
func NewSSEHub() *SSEHub {
	h := &SSEHub{
		clients:  make(map[string]map[*sseClient]struct{}),
		incoming: make(chan sseEvent, 256),
	}

	go h.run()
	return h
}

func (h *SSEHub) run() {
	for ev := range h.incoming {
		h.mu.RLock()
		subs := h.clients[ev.Channel]
		for c := range subs {
			select {
			case c.ch <- ev:
			default:
				// slow / backed-up clients drop events

			}
		}
		h.mu.RUnlock()
	}
}

// Subscribe returns a client subscribed to a channel.
func (h *SSEHub) Subscribe(channel string) *sseClient {
	c := &sseClient{
		ch:   make(chan sseEvent, 16),
		done: make(chan struct{}),
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[channel] == nil {
		h.clients[channel] = make(map[*sseClient]struct{})
	}
	h.clients[channel][c] = struct{}{}
	return c
}

// Unsubscribe Unsusbscribe removes a client from a channel and closes its done channel.
func (h *SSEHub) Unsubscribe(channel string, c *sseClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs := h.clients[channel]
	if subs == nil {
		return
	}

	delete(subs, c)
	close(c.done)
	if len(subs) == 0 {
		delete(h.clients, channel)
	}
}

// Publish JSON-encodes payload and broadcasts it to all subscribers
func (h *SSEHub) Publish(channel, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[sse] marshal error: %v", err)
		return
	}
	h.incoming <- sseEvent{
		Channel: channel,
		Event:   event,
		Data:    data,
	}
}
