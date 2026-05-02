package task

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/user/aria2-aio/internal/rpc"
	"github.com/user/aria2-aio/internal/store"
	"github.com/user/aria2-aio/internal/ws"
)

type InstanceTaskManager struct {
	instanceID    string
	rpcClient     *rpc.Aria2Client
	eventListener *rpc.EventListener
	hub           *ws.Hub
	store         *store.TaskRepo
	pollInterval  time.Duration
	cancel        context.CancelFunc
}

type TaskTracker struct {
	mu       sync.RWMutex
	managers map[string]*InstanceTaskManager
	hub      *ws.Hub
	store    *store.TaskRepo
}

func NewTaskTracker(hub *ws.Hub, store *store.TaskRepo) *TaskTracker {
	return &TaskTracker{
		managers: make(map[string]*InstanceTaskManager),
		hub:      hub,
		store:    store,
	}
}

func (t *TaskTracker) Register(instanceID string, rpcClient *rpc.Aria2Client, eventListener *rpc.EventListener, pollIntervalSeconds int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.managers[instanceID]; exists {
		t.Unregister(instanceID)
	}

	ctx, cancel := context.WithCancel(context.Background())

	mgr := &InstanceTaskManager{
		instanceID:    instanceID,
		rpcClient:     rpcClient,
		eventListener: eventListener,
		hub:           t.hub,
		store:         t.store,
		pollInterval:  time.Duration(pollIntervalSeconds) * time.Second,
		cancel:        cancel,
	}

	t.managers[instanceID] = mgr

	go mgr.pollLoop(ctx)
	go mgr.eventLoop(ctx)
}

func (t *TaskTracker) Unregister(instanceID string) {
	t.mu.Lock()
	mgr, exists := t.managers[instanceID]
	if exists {
		delete(t.managers, instanceID)
	}
	t.mu.Unlock()

	if mgr != nil && mgr.cancel != nil {
		mgr.cancel()
	}
}

func (t *TaskTracker) UnregisterAll() {
	t.mu.Lock()
	ids := make([]string, 0, len(t.managers))
	for id := range t.managers {
		ids = append(ids, id)
	}
	t.mu.Unlock()

	for _, id := range ids {
		t.Unregister(id)
	}
}

func (mgr *InstanceTaskManager) pollLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(mgr.pollInterval):
		}

		mgr.pollAndBroadcast(ctx)
	}
}

func (mgr *InstanceTaskManager) pollAndBroadcast(ctx context.Context) {
	pollCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys := []string{
		"gid", "status", "totalLength", "completedLength",
		"downloadSpeed", "uploadSpeed", "uploadLength",
		"numSeeders", "connections", "errorCode", "errorMessage",
		"bittorrent", "files",
	}

	active, err := mgr.rpcClient.TellActive(pollCtx, keys)
	if err != nil {
		return
	}

	waiting, err := mgr.rpcClient.TellWaiting(pollCtx, 0, 100, keys)
	if err != nil {
		return
	}

	allTasks := make([]TaskProgress, 0, len(active)+len(waiting))
	for _, ds := range active {
		allTasks = append(allTasks, TaskProgressFromDownloadStatus(&ds))
	}
	for _, ds := range waiting {
		allTasks = append(allTasks, TaskProgressFromDownloadStatus(&ds))
	}

	mgr.hub.Broadcast(ws.Message{
		Type:       "task_progress",
		InstanceID: mgr.instanceID,
		Data:       allTasks,
	})

	// Scavenge completed/error tasks from aria2's stopped list that aren't in history yet.
	// This catches tasks that completed before the EventListener connected.
	mgr.scavengeCompletedTasks(pollCtx)
}

func (mgr *InstanceTaskManager) scavengeCompletedTasks(ctx context.Context) {
	stopped, err := mgr.rpcClient.TellStopped(ctx, 0, 50, nil)
	if err != nil {
		return
	}

	for _, ds := range stopped {
		if ds.Status != "complete" && ds.Status != "error" {
			continue
		}

		exists, err := mgr.store.ExistsByGID(mgr.instanceID, ds.GID)
		if err != nil || exists {
			continue
		}

		record := TaskRecordFromDownloadStatus(mgr.instanceID, &ds)
		if err := mgr.store.Create(record); err != nil {
			log.Printf("TaskTracker %s: failed to record completed task %s: %v", mgr.instanceID, ds.GID, err)
			continue
		}

		mgr.hub.Broadcast(ws.Message{
			Type:       "task_completed",
			InstanceID: mgr.instanceID,
			Data:       record,
		})

		// Remove from aria2's stopped list to prevent memory bloat
		cleanCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		mgr.rpcClient.RemoveDownloadResult(cleanCtx, ds.GID)
		cancel()
	}
}

func (mgr *InstanceTaskManager) eventLoop(ctx context.Context) {
	events := mgr.eventListener.Events()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			mgr.hub.Broadcast(ws.Message{
				Type:       "task_event",
				InstanceID: mgr.instanceID,
				Data: map[string]string{
					"event": string(event.EventType),
					"gid":   event.GID,
				},
			})

			// On completion/error, record to history
			if event.EventType == rpc.EventDownloadComplete || event.EventType == rpc.EventDownloadError || event.EventType == rpc.EventBtDownloadComplete {
				mgr.recordCompletedTask(ctx, event.GID)
			}
		}
	}
}

func (mgr *InstanceTaskManager) recordCompletedTask(ctx context.Context, gid string) {
	detailCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ds, err := mgr.rpcClient.TellStatus(detailCtx, gid, nil)
	if err != nil {
		log.Printf("TaskTracker %s: tellStatus for %s failed: %v", mgr.instanceID, gid, err)
		return
	}

	// Check if already recorded (avoid duplicates)
	exists, _ := mgr.store.ExistsByGID(mgr.instanceID, gid)
	if exists {
		return
	}

	record := TaskRecordFromDownloadStatus(mgr.instanceID, ds)
	if err := mgr.store.Create(record); err != nil {
		log.Printf("TaskTracker %s: failed to record task %s: %v", mgr.instanceID, gid, err)
		return
	}

	mgr.hub.Broadcast(ws.Message{
		Type:       "task_completed",
		InstanceID: mgr.instanceID,
		Data:       record,
	})

	// Clean up from aria2's stopped list
	cleanCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	mgr.rpcClient.RemoveDownloadResult(cleanCtx, gid)
	cancel()
}