package realtime

import "sync"

type Hub struct {
	clients map[string]map[*Client]struct{}
	mux     sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Register(client *Client) {
	h.mux.Lock()
	defer h.mux.Unlock()
	if h.clients[client.userID] == nil {
		h.clients[client.userID] = make(map[*Client]struct{})
	}

	h.clients[client.userID][client] = struct{}{}
}

func (h *Hub) Unregister(client *Client) {
	h.mux.Lock()
	defer h.mux.Unlock()
	delete(h.clients[client.userID], client)

	if len(h.clients[client.userID]) == 0 {
		delete(h.clients, client.userID)
	}
}

func (h *Hub) ConnectionCount(userID string) int {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return len(h.clients[userID])
}
