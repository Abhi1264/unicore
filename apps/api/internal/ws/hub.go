package ws

import (
	"sync"

	"github.com/google/uuid"
)

// Hub is an in-memory tenant-scoped pub/sub for announcements.
type Hub struct {
	mu   sync.RWMutex
	subs map[uuid.UUID]map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[uuid.UUID]map[chan []byte]struct{})}
}

func (h *Hub) Subscribe(tenantID uuid.UUID) chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[tenantID] == nil {
		h.subs[tenantID] = make(map[chan []byte]struct{})
	}
	h.subs[tenantID][ch] = struct{}{}
	return ch
}

func (h *Hub) Unsubscribe(tenantID uuid.UUID, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.subs[tenantID]; ok {
		if _, ok := m[ch]; ok {
			delete(m, ch)
			close(ch)
		}
		if len(m) == 0 {
			delete(h.subs, tenantID)
		}
	}
}

func (h *Hub) Publish(tenantID uuid.UUID, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[tenantID] {
		select {
		case ch <- payload:
		default:
			// drop if slow consumer
		}
	}
}
