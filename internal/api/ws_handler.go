package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/user/aria2-aio/internal/ws"
)

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("WebSocket accept failed: %v", err)
		return
	}

	client := ws.NewClient(conn, s.hub)
	s.hub.RegisterClient(client)

	// Send initial state snapshot
	go s.sendInitialSnapshot(client)

	// Start write pump
	go client.WritePump()

	// Read pump - keep connection alive, discard client messages
	for {
		_, _, err := conn.Read(context.Background())
		if err != nil {
			break
		}
	}

	s.hub.UnregisterClient(client)
}

func (s *Server) sendInitialSnapshot(client *ws.Client) {
	instances, err := s.store.Instances().List()
	if err != nil {
		return
	}

	for _, inst := range instances {
		status := inst.Status
		if s.manager.IsActive(inst.ID) {
			status = "running"
		}
		msg := ws.Message{
			Type:       "instance_status",
			InstanceID: inst.ID,
			Data:       map[string]string{"status": status},
		}
		select {
		case client.Send <- msg.ToJSON():
		default:
			return
		}
	}

	activeInstances := s.manager.ListActive()
	for id, ai := range activeInstances {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		tasks, err := ai.RPCClient.TellActive(ctx, nil)
		cancel()
		if err != nil {
			continue
		}

		var taskData []map[string]any
		for _, ds := range tasks {
			taskData = append(taskData, downloadStatusToMap(&ds))
		}

		msg := ws.Message{
			Type:       "task_progress",
			InstanceID: id,
			Data:       taskData,
		}
		select {
		case client.Send <- msg.ToJSON():
		default:
			return
		}
	}
}