package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/user/aria2-aio/internal/rpc"
	"github.com/user/aria2-aio/internal/store"
)

func (s *Server) ListActiveTasks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	active, exists := s.manager.GetActive(id)
	if !exists {
		writeError(w, 400, "instance is not running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasks, err := active.RPCClient.TellActive(ctx, nil)
	if err != nil {
		writeError(w, 500, "rpc call failed: "+err.Error())
		return
	}

	// Convert to task progress format
	progress := make([]map[string]any, 0, len(tasks))
	for _, ds := range tasks {
		progress = append(progress, downloadStatusToMap(&ds))
	}
	writeJSON(w, 200, progress)
}

func (s *Server) ListWaitingTasks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	active, exists := s.manager.GetActive(id)
	if !exists {
		writeError(w, 400, "instance is not running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasks, err := active.RPCClient.TellWaiting(ctx, 0, 100, nil)
	if err != nil {
		writeError(w, 500, "rpc call failed: "+err.Error())
		return
	}

	var progress = make([]map[string]any, 0)
	for _, ds := range tasks {
		progress = append(progress, downloadStatusToMap(&ds))
	}
	writeJSON(w, 200, progress)
}

func (s *Server) ListStoppedTasks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	active, exists := s.manager.GetActive(id)
	if !exists {
		writeError(w, 400, "instance is not running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasks, err := active.RPCClient.TellStopped(ctx, 0, 100, nil)
	if err != nil {
		writeError(w, 500, "rpc call failed: "+err.Error())
		return
	}

	var progress = make([]map[string]any, 0)
	for _, ds := range tasks {
		progress = append(progress, downloadStatusToMap(&ds))
	}
	writeJSON(w, 200, progress)
}

func (s *Server) AddTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	active, exists := s.manager.GetActive(id)
	if !exists {
		writeError(w, 400, "instance is not running")
		return
	}

	var req struct {
		Type    string            `json:"type"`    // uri, torrent, metalink
		URIs    []string          `json:"uris"`    // for uri type
		Torrent string            `json:"torrent"` // base64 encoded torrent
		Metalink string           `json:"metalink"` // base64 encoded metalink
		Options map[string]string `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch req.Type {
	case "uri":
		if len(req.URIs) == 0 {
			writeError(w, 400, "uris is required for uri type")
			return
		}
		gid, err := active.RPCClient.AddURI(ctx, req.URIs, req.Options)
		if err != nil {
			writeError(w, 500, "add uri failed: "+err.Error())
			return
		}
		writeJSON(w, 201, map[string]string{"gid": gid})

	case "torrent":
		if req.Torrent == "" {
			writeError(w, 400, "torrent is required for torrent type")
			return
		}
		gid, err := active.RPCClient.AddTorrent(ctx, req.Torrent, req.URIs, req.Options)
		if err != nil {
			writeError(w, 500, "add torrent failed: "+err.Error())
			return
		}
		writeJSON(w, 201, map[string]string{"gid": gid})

	case "metalink":
		if req.Metalink == "" {
			writeError(w, 400, "metalink is required for metalink type")
			return
		}
		gids, err := active.RPCClient.AddMetalink(ctx, req.Metalink, req.Options)
		if err != nil {
			writeError(w, 500, "add metalink failed: "+err.Error())
			return
		}
		writeJSON(w, 201, map[string]any{"gids": gids})

	default:
		writeError(w, 400, "invalid type: must be uri, torrent, or metalink")
	}
}

func (s *Server) GetTaskStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	gid := r.PathValue("gid")
	active, exists := s.manager.GetActive(id)
	if !exists {
		writeError(w, 400, "instance is not running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := active.RPCClient.TellStatus(ctx, gid, nil)
	if err != nil {
		writeError(w, 500, "rpc call failed: "+err.Error())
		return
	}

	writeJSON(w, 200, downloadStatusToMap(status))
}

func (s *Server) PauseTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	gid := r.PathValue("gid")
	active, exists := s.manager.GetActive(id)
	if !exists {
		writeError(w, 400, "instance is not running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := active.RPCClient.Pause(ctx, gid)
	if err != nil {
		writeError(w, 500, "pause failed: "+err.Error())
		return
	}

	writeJSON(w, 200, map[string]string{"status": "paused"})
}

func (s *Server) UnpauseTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	gid := r.PathValue("gid")
	active, exists := s.manager.GetActive(id)
	if !exists {
		writeError(w, 400, "instance is not running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := active.RPCClient.Unpause(ctx, gid)
	if err != nil {
		writeError(w, 500, "unpause failed: "+err.Error())
		return
	}

	writeJSON(w, 200, map[string]string{"status": "active"})
}

func (s *Server) RemoveTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	gid := r.PathValue("gid")
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	active, exists := s.manager.GetActive(id)
	if !exists {
		writeError(w, 400, "instance is not running")
		return
	}

	// If delete_files, grab the download status first to know file paths
	if deleteFiles {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ds, err := active.RPCClient.TellStatus(ctx, gid, []string{"gid", "dir", "files"})
		cancel()
		if err == nil && ds != nil {
			deleteFilesFromRecord(ds.Dir, ds.Files)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := active.RPCClient.Remove(ctx, gid)
	if err != nil {
		_, err2 := active.RPCClient.ForceRemove(ctx, gid)
		if err2 != nil {
			writeError(w, 500, "remove failed: "+err.Error())
			return
		}
	}

	writeJSON(w, 200, map[string]string{"status": "removed"})
}

func (s *Server) ListHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	result, err := s.store.Tasks().ListByInstance(id, page, perPage)
	if err != nil {
		writeError(w, 500, "query failed: "+err.Error())
		return
	}

	writeJSON(w, 200, result)
}

func (s *Server) GetHistoryRecord(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	gid := r.PathValue("gid")

	record, err := s.store.Tasks().GetByGID(id, gid)
	if err != nil {
		writeError(w, 404, "record not found")
		return
	}

	writeJSON(w, 200, record)
}

func (s *Server) DeleteHistoryRecord(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	gid := r.PathValue("gid")
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	record, err := s.store.Tasks().GetByGID(id, gid)
	if err != nil {
		writeError(w, 404, "record not found")
		return
	}

	if deleteFiles && record != nil {
		deleteFilesFromHistoryRecord(record)
	}

	if err := s.store.Tasks().DeleteByGID(id, gid); err != nil {
		writeError(w, 500, "delete failed: "+err.Error())
		return
	}

	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) GetGlobalStats(w http.ResponseWriter, r *http.Request) {
	activeInstances := s.manager.ListActive()

	totalDownloadSpeed := int64(0)
	totalUploadSpeed := int64(0)
	numActive := 0

	for _, ai := range activeInstances {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		stat, err := ai.RPCClient.GetGlobalStat(ctx)
		cancel()
		if err != nil {
			continue
		}
		totalDownloadSpeed += rpc.ParseInt64(stat.DownloadSpeed)
		totalUploadSpeed += rpc.ParseInt64(stat.UploadSpeed)
		numActive += int(rpc.ParseInt64(stat.NumActive))
	}

	writeJSON(w, 200, map[string]any{
		"total_download_speed": totalDownloadSpeed,
		"total_upload_speed":   totalUploadSpeed,
		"num_active_downloads": numActive,
		"num_running_instances": len(activeInstances),
	})
}

func (s *Server) GetVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{
		"version": "0.1.0",
		"name":    "aria2-aio",
	})
}

func downloadStatusToMap(ds *rpc.DownloadStatus) map[string]any {
	return map[string]any{
		"gid":              ds.GID,
		"status":           ds.Status,
		"name":             rpc.DownloadName(ds),
		"total_length":     rpc.ParseInt64(ds.TotalLength),
		"completed_length": rpc.ParseInt64(ds.CompletedLength),
		"download_speed":   rpc.ParseInt64(ds.DownloadSpeed),
		"upload_speed":     rpc.ParseInt64(ds.UploadSpeed),
		"upload_length":    rpc.ParseInt64(ds.UploadLength),
		"connections":      rpc.ParseInt64(ds.Connections),
		"num_seeders":      rpc.ParseInt64(ds.NumSeeders),
		"error_code":       ds.ErrorCode,
		"error_message":    ds.ErrorMessage,
		"dir":              ds.Dir,
	}
}

// deleteFilesFromRecord deletes downloaded files given a download directory and file list.
// For each file, if the path starts with dir, it tries to remove the file.
// After removing all individual files, it tries to remove the parent directory if empty.
func deleteFilesFromRecord(dir string, files []rpc.File) {
	if dir == "" || len(files) == 0 {
		return
	}

	var lastDir string
	for _, f := range files {
		if f.Path == "" {
			continue
		}
		// Only delete files that are under the download directory
		if !filepath.HasPrefix(f.Path, dir) {
			continue
		}
		if err := os.Remove(f.Path); err != nil {
			fmt.Printf("deleteFiles: failed to remove %s: %v\n", f.Path, err)
		}
		lastDir = filepath.Dir(f.Path)
	}

	// Try to remove empty parent directories
	for d := lastDir; d != "" && d != dir; d = filepath.Dir(d) {
		if err := os.Remove(d); err != nil {
			break // not empty or doesn't exist, stop
		}
	}
}

// deleteFilesFromHistoryRecord deletes files for a completed task using stored history data.
func deleteFilesFromHistoryRecord(record *store.TaskRecord) {
	if record.Dir == "" || record.FilesJSON == "" {
		return
	}

	var files []rpc.File
	if err := json.Unmarshal([]byte(record.FilesJSON), &files); err != nil {
		fmt.Printf("deleteFilesFromHistoryRecord: failed to parse files_json: %v\n", err)
		return
	}

	deleteFilesFromRecord(record.Dir, files)
}