package api

import (
	"encoding/json"
	"net/http"

	"github.com/user/aria2-aio/internal/instance"
	"github.com/user/aria2-aio/internal/store"
	"github.com/user/aria2-aio/internal/task"
	"github.com/user/aria2-aio/internal/ws"
)

type Server struct {
	manager    *instance.InstanceManager
	store      *store.DB
	tracker    *task.TaskTracker
	hub        *ws.Hub
}

func NewServer(manager *instance.InstanceManager, store *store.DB, tracker *task.TaskTracker, hub *ws.Hub) *Server {
	return &Server{
		manager: manager,
		store:   store,
		tracker: tracker,
		hub:     hub,
	}
}

func (s *Server) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/instances", s.ListInstances)
	mux.HandleFunc("POST /api/v1/instances", s.CreateInstance)
	mux.HandleFunc("GET /api/v1/instances/{id}", s.GetInstance)
	mux.HandleFunc("PUT /api/v1/instances/{id}", s.UpdateInstance)
	mux.HandleFunc("DELETE /api/v1/instances/{id}", s.DeleteInstance)
	mux.HandleFunc("POST /api/v1/instances/{id}/start", s.StartInstance)
	mux.HandleFunc("POST /api/v1/instances/{id}/stop", s.StopInstance)
	mux.HandleFunc("POST /api/v1/instances/{id}/restart", s.RestartInstance)

	mux.HandleFunc("GET /api/v1/instances/{id}/tasks/active", s.ListActiveTasks)
	mux.HandleFunc("GET /api/v1/instances/{id}/tasks/waiting", s.ListWaitingTasks)
	mux.HandleFunc("GET /api/v1/instances/{id}/tasks/stopped", s.ListStoppedTasks)
	mux.HandleFunc("POST /api/v1/instances/{id}/tasks", s.AddTask)
	mux.HandleFunc("GET /api/v1/instances/{id}/tasks/{gid}", s.GetTaskStatus)
	mux.HandleFunc("POST /api/v1/instances/{id}/tasks/{gid}/pause", s.PauseTask)
	mux.HandleFunc("POST /api/v1/instances/{id}/tasks/{gid}/unpause", s.UnpauseTask)
	mux.HandleFunc("DELETE /api/v1/instances/{id}/tasks/{gid}", s.RemoveTask)

	mux.HandleFunc("GET /api/v1/instances/{id}/history", s.ListHistory)
	mux.HandleFunc("GET /api/v1/instances/{id}/history/{gid}", s.GetHistoryRecord)
	mux.HandleFunc("DELETE /api/v1/instances/{id}/history/{gid}", s.DeleteHistoryRecord)

	mux.HandleFunc("GET /api/v1/stats", s.GetGlobalStats)
	mux.HandleFunc("GET /api/v1/version", s.GetVersion)
	mux.HandleFunc("GET /api/v1/ws", s.HandleWebSocket)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) ListInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := s.store.Instances().List()
	if err != nil {
		writeError(w, 500, "failed to list instances")
		return
	}
	if instances == nil {
		instances = []*store.Instance{}
	}
	// Enrich with runtime status
	for _, inst := range instances {
		if s.manager.IsActive(inst.ID) {
			inst.Status = "running"
			pid, _ := s.manager.GetActive(inst.ID)
			if pid != nil {
				inst.PID = pid.Instance.PID
			}
		}
	}
	writeJSON(w, 200, instances)
}

func (s *Server) CreateInstance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string            `json:"name"`
		Dir     string            `json:"dir"`
		Options map[string]string `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, 400, "name is required")
		return
	}

	inst, err := s.manager.Create(req.Name, req.Dir, req.Options)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	s.hub.Broadcast(ws.Message{Type: "instance_status", InstanceID: inst.ID, Data: map[string]string{"status": "created"}})
	writeJSON(w, 201, inst)
}

func (s *Server) GetInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := s.store.Instances().GetByID(id)
	if err != nil {
		writeError(w, 404, "instance not found")
		return
	}

	if s.manager.IsActive(id) {
		inst.Status = "running"
	}

	writeJSON(w, 200, inst)
}

func (s *Server) UpdateInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name    string            `json:"name"`
		Dir     string            `json:"dir"`
		Options map[string]string `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}

	inst, err := s.manager.Update(id, req.Name, req.Dir, req.Options)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	writeJSON(w, 200, inst)
}

func (s *Server) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.Delete(id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	s.hub.Broadcast(ws.Message{Type: "instance_status", InstanceID: id, Data: map[string]string{"status": "deleted"}})
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) StartInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := s.manager.Start(id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	// Register with task tracker
	active, _ := s.manager.GetActive(id)
	if active != nil {
		s.tracker.Register(id, active.RPCClient, active.Listener, 2)
	}

	s.hub.Broadcast(ws.Message{Type: "instance_status", InstanceID: id, Data: map[string]string{"status": "running"}})
	writeJSON(w, 200, inst)
}

func (s *Server) StopInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Unregister tracker first to stop polling before aria2c shuts down
	s.tracker.Unregister(id)

	inst, err := s.manager.Stop(id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	s.hub.Broadcast(ws.Message{Type: "instance_status", InstanceID: id, Data: map[string]string{"status": "stopped"}})
	writeJSON(w, 200, inst)
}

func (s *Server) RestartInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.tracker.Unregister(id)
	inst, err := s.manager.Restart(id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	active, _ := s.manager.GetActive(id)
	if active != nil {
		s.tracker.Register(id, active.RPCClient, active.Listener, 2)
	}

	s.hub.Broadcast(ws.Message{Type: "instance_status", InstanceID: id, Data: map[string]string{"status": "running"}})
	writeJSON(w, 200, inst)
}