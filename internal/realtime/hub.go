package realtime

import (
	"sync"

	"github.com/coder/websocket"
)

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

func (h *Hub) SendToUser(userID string, data []byte) {
	clients := h.clientsForUser(userID)
	for _, client := range clients {
		if client.Send(data) {
			continue
		}
		h.Unregister(client)
		client.Close(websocket.StatusPolicyViolation, "slow client")
	}
}

func (h *Hub) SendToUsers(userIDs []string, data []byte) {
	for _, userID := range userIDs {
		h.SendToUser(userID, data)
	}
}

func (h *Hub) Shutdown() {
	clients := h.allClients()
	for _, client := range clients {
		client.Close(websocket.StatusNormalClosure, "server shutdown")
	}
}

func (h *Hub) clientsForUser(userID string) []*Client {
	h.mux.RLock()
	defer h.mux.RUnlock()
	clients := make([]*Client, 0, len(h.clients[userID]))
	for client := range h.clients[userID] {
		clients = append(clients, client)
	}
	return clients
}

func (h *Hub) allClients() []*Client {
	h.mux.Lock()
	defer h.mux.Unlock()
	clients := make([]*Client, 0)
	for _, userClients := range h.clients {
		for client := range userClients {
			clients = append(clients, client)
		}
	}
	h.clients = make(map[string]map[*Client]struct{})
	return clients
}
