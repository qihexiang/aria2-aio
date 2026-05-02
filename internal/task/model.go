package task

import (
	"encoding/json"

	"github.com/user/aria2-aio/internal/rpc"
	"github.com/user/aria2-aio/internal/store"
)

type TaskProgress struct {
	GID             string `json:"gid"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	TotalLength     int64  `json:"total_length"`
	CompletedLength int64  `json:"completed_length"`
	DownloadSpeed   int64  `json:"download_speed"`
	UploadSpeed     int64  `json:"upload_speed"`
	UploadLength    int64  `json:"upload_length"`
	Connections     int    `json:"connections"`
	NumSeeders      int    `json:"num_seeders"`
}

func TaskProgressFromDownloadStatus(ds *rpc.DownloadStatus) TaskProgress {
	return TaskProgress{
		GID:             ds.GID,
		Name:            rpc.DownloadName(ds),
		Status:          ds.Status,
		TotalLength:     rpc.ParseInt64(ds.TotalLength),
		CompletedLength: rpc.ParseInt64(ds.CompletedLength),
		DownloadSpeed:   rpc.ParseInt64(ds.DownloadSpeed),
		UploadSpeed:     rpc.ParseInt64(ds.UploadSpeed),
		UploadLength:    rpc.ParseInt64(ds.UploadLength),
		Connections:     int(rpc.ParseInt64(ds.Connections)),
		NumSeeders:      int(rpc.ParseInt64(ds.NumSeeders)),
	}
}

func TaskRecordFromDownloadStatus(instanceID string, ds *rpc.DownloadStatus) *store.TaskRecord {
	var uris []string
	for _, f := range ds.Files {
		for _, u := range f.URIs {
			uris = append(uris, u.URI)
		}
	}
	urisJSON, _ := json.Marshal(uris)

	filesJSON, _ := json.Marshal(ds.Files)

	infoHash := ""
	if ds.Bittorrent != nil && ds.Bittorrent.Info != nil {
		infoHash = ds.Bittorrent.Info.Name
	}

	return &store.TaskRecord{
		InstanceID:      instanceID,
		GID:             ds.GID,
		Name:            rpc.DownloadName(ds),
		URIs:            string(urisJSON),
		Dir:             ds.Dir,
		FilesJSON:       string(filesJSON),
		TotalLength:     rpc.ParseInt64(ds.TotalLength),
		CompletedLength: rpc.ParseInt64(ds.CompletedLength),
		DownloadSpeed:   rpc.ParseInt64(ds.DownloadSpeed),
		UploadLength:    rpc.ParseInt64(ds.UploadLength),
		Status:          ds.Status,
		ErrorCode:       int(rpc.ParseInt64(ds.ErrorCode)),
		ErrorMessage:    ds.ErrorMessage,
		InfoHash:        infoHash,
	}
}