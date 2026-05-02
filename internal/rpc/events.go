package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type EventType string

const (
	EventDownloadStart      EventType = "onDownloadStart"
	EventDownloadPause      EventType = "onDownloadPause"
	EventDownloadStop       EventType = "onDownloadStop"
	EventDownloadComplete   EventType = "onDownloadComplete"
	EventDownloadError      EventType = "onDownloadError"
	EventBtDownloadComplete EventType = "onBtDownloadComplete"
)

type Aria2Event struct {
	EventType EventType `json:"type"`
	GID       string    `json:"gid"`
}

type EventListener struct {
	client  *Aria2Client
	wsURL   string
	conn    *websocket.Conn
	events  chan Aria2Event
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
}

func NewEventListener(client *Aria2Client) *EventListener {
	port := 0
	_, host, err := net.SplitHostPort(client.baseURL[7:])
	if err == nil {
		fmt.Sscanf(host, "%d", &port)
	}
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/jsonrpc", port)

	return &EventListener{
		client: client,
		wsURL:  wsURL,
		events: make(chan Aria2Event, 64),
	}
}

func (l *EventListener) Start(ctx context.Context) {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return
	}
	l.running = true
	l.mu.Unlock()

	ctx, l.cancel = context.WithCancel(ctx)
	go l.listen(ctx)
}

func (l *EventListener) Stop() {
	l.mu.Lock()
	l.running = false
	l.mu.Unlock()

	if l.cancel != nil {
		l.cancel()
	}
	if l.conn != nil {
		l.conn.Close(websocket.StatusNormalClosure, "stopping")
	}
}

func (l *EventListener) Events() <-chan Aria2Event {
	return l.events
}

func (l *EventListener) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := l.connect(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// Suppress error log during shutdown
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				continue
			}
		}

		l.mu.Lock()
		l.conn = conn
		l.mu.Unlock()

		err = l.readLoop(ctx, conn)
		conn.Close(websocket.StatusNormalClosure, "read loop ended")

		select {
		case <-ctx.Done():
			return
		default:
		}
		// Only log if still running (not during shutdown)
		if err != nil {
			l.mu.Lock()
			stillRunning := l.running
			l.mu.Unlock()
			if stillRunning {
				log.Printf("EventListener %s: reconnecting after: %v", l.client.instanceID, err)
			}
		}
	}
}

func (l *EventListener) connect(ctx context.Context) (*websocket.Conn, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(connectCtx, l.wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	// aria2 WebSocket automatically sends notifications when connected.
	// No subscription message needed.
	return conn, nil
}

func (l *EventListener) readLoop(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read without timeout - aria2 events are infrequent
		_, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var notification map[string]any
		if err := json.Unmarshal(data, &notification); err != nil {
			continue
		}

		method, _ := notification["method"].(string)
		params, _ := notification["params"].([]any)

		if len(params) < 1 {
			continue
		}

		gid, _ := params[0].(string)

		var eventType EventType
		switch method {
		case "aria2.onDownloadStart":
			eventType = EventDownloadStart
		case "aria2.onDownloadPause":
			eventType = EventDownloadPause
		case "aria2.onDownloadStop":
			eventType = EventDownloadStop
		case "aria2.onDownloadComplete":
			eventType = EventDownloadComplete
		case "aria2.onDownloadError":
			eventType = EventDownloadError
		case "aria2.onBtDownloadComplete":
			eventType = EventBtDownloadComplete
		}

		if eventType != "" && gid != "" {
			select {
			case l.events <- Aria2Event{EventType: eventType, GID: gid}:
			default:
				log.Printf("EventListener %s: events channel full, dropping %s/%s",
					l.client.instanceID, eventType, gid)
			}
		}
	}
}