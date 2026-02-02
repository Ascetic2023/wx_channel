package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"wx_channel/hub_server/database"
	"wx_channel/hub_server/models"

	"github.com/gorilla/websocket"
)

type Hub struct {
	Clients    map[string]*Client
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  10 * 1024 * 1024, // 10MB 读缓冲
			WriteBufferSize: 10 * 1024 * 1024, // 10MB 写缓冲
		},
	}
}

func (h *Hub) Run() {
	// 启动僵尸连接清理器
	go h.cleanupStaleConnections()

	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if old, ok := h.Clients[client.ID]; ok {
				old.Conn.Close()
			}
			h.Clients[client.ID] = client
			h.mu.Unlock()

			log.Printf("Client connected: %s", client.ID)
			// DB: Mark as online
			database.UpsertNode(&models.Node{
				ID:       client.ID,
				Status:   "online",
				LastSeen: time.Now(),
			})

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				client.Conn.Close()

				log.Printf("Client disconnected: %s", client.ID)
				// DB: Mark as offline
				database.UpdateNodeStatus(client.ID, "offline")
			}
			h.mu.Unlock()
		}
	}
}

// cleanupStaleConnections 清理僵尸连接
func (h *Hub) cleanupStaleConnections() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		staleClients := []*Client{}
		threshold := time.Now().Add(-2 * time.Minute) // 2分钟无心跳视为僵尸连接

		for _, client := range h.Clients {
			client.mu.Lock()
			if client.LastSeen.Before(threshold) {
				staleClients = append(staleClients, client)
			}
			client.mu.Unlock()
		}
		h.mu.RUnlock()

		// 清理僵尸连接
		for _, client := range staleClients {
			log.Printf("清理僵尸连接: %s (最后心跳: %v)", client.ID, client.LastSeen)
			h.Unregister <- client
		}
	}
}

func (h *Hub) RemoveClient(id string) {
	h.mu.Lock()
	if c, ok := h.Clients[id]; ok {
		c.Conn.Close()
		delete(h.Clients, id)
	}
	h.mu.Unlock()
}

// GetClient safely retrieves a client by ID
func (h *Hub) GetClient(id string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Clients[id]
}

func (h *Hub) Call(userID uint, clientID string, action string, data interface{}, timeout time.Duration) (ResponsePayload, error) {
	h.mu.RLock()
	c, ok := h.Clients[clientID]
	h.mu.RUnlock()

	if !ok {
		return ResponsePayload{}, fmt.Errorf("client offline")
	}

	reqID := fmt.Sprintf("hub-%d", time.Now().UnixNano())
	payloadData, _ := json.Marshal(data)
	cmd := CommandPayload{Action: action, Data: payloadData}
	cmdData, _ := json.Marshal(cmd)

	// DB: Create Task
	task := &models.Task{
		Type:    action,
		NodeID:  clientID,
		UserID:  userID,
		Payload: string(payloadData),
		Status:  "pending",
	}
	database.CreateTask(task)

	msg := CloudMessage{
		ID:        reqID,
		Type:      MsgTypeCommand,
		ClientID:  "hub-server",
		Payload:   cmdData,
		Timestamp: time.Now().Unix(),
	}

	respChan := make(chan ResponsePayload, 1)
	c.respMu.Lock()
	c.respChannels[reqID] = respChan
	c.respMu.Unlock()

	// 确保清理资源
	defer func() {
		c.respMu.Lock()
		delete(c.respChannels, reqID)
		c.respMu.Unlock()
	}()

	msgData, _ := json.Marshal(msg)

	if err := c.WriteMessage(msgData); err != nil {
		database.UpdateTaskResult(task.ID, "failed", "", err.Error())
		return ResponsePayload{}, fmt.Errorf("发送消息失败: %w", err)
	}

	select {
	case resp, ok := <-respChan:
		if !ok {
			database.UpdateTaskResult(task.ID, "failed", "", "响应通道已关闭")
			return ResponsePayload{}, fmt.Errorf("响应通道已关闭")
		}
		resBytes, _ := json.Marshal(resp.Data)
		status := "success"
		if !resp.Success {
			status = "failed"
		}
		database.UpdateTaskResult(task.ID, status, string(resBytes), resp.Error)
		return resp, nil
	case <-time.After(timeout):
		database.UpdateTaskResult(task.ID, "timeout", "", "request timeout")
		return ResponsePayload{}, fmt.Errorf("请求超时")
	}
}

func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = r.URL.Query().Get("client_id")
	}
	if clientID == "" {
		http.Error(w, "X-Client-ID required", 400)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	client := NewClient(clientID, conn, h)
	h.Register <- client

	// Start reading (blocking until disconnect)
	go client.ReadPump()
}
