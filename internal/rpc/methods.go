package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

type DownloadStatus struct {
	GID             string `json:"gid"`
	Status          string `json:"status"`
	TotalLength     string `json:"totalLength"`
	CompletedLength string `json:"completedLength"`
	DownloadSpeed   string `json:"downloadSpeed"`
	UploadSpeed     string `json:"uploadSpeed"`
	UploadLength    string `json:"uploadLength"`
	NumSeeders      string `json:"numSeeders"`
	Connections     string `json:"connections"`
	ErrorCode       string `json:"errorCode"`
	ErrorMessage    string `json:"errorMessage"`
	Dir             string `json:"dir"`
	Files           []File `json:"files"`
	Bittorrent      *BT    `json:"bittorrent"`
}

type File struct {
	Index           string `json:"index"`
	Path            string `json:"path"`
	Length          string `json:"length"`
	CompletedLength string `json:"completedLength"`
	Selected        string `json:"selected"`
	URIs            []URI  `json:"uris"`
}

type URI struct {
	URI    string `json:"uri"`
	Status string `json:"status"`
}

type BT struct {
	Info *BTInfo `json:"info"`
}

type BTInfo struct {
	Name string `json:"name"`
}

type GlobalStat struct {
	DownloadSpeed   string `json:"downloadSpeed"`
	UploadSpeed     string `json:"uploadSpeed"`
	NumActive       string `json:"numActive"`
	NumWaiting      string `json:"numWaiting"`
	NumStopped      string `json:"numStopped"`
	NumStoppedTotal string `json:"numStoppedTotal"`
}

type VersionInfo struct {
	Version    string `json:"version"`
	EnabledFeatures []string `json:"enabledFeatures"`
}

var statusKeys = []string{
	"gid", "status", "totalLength", "completedLength",
	"downloadSpeed", "uploadSpeed", "uploadLength",
	"numSeeders", "connections", "errorCode", "errorMessage",
	"dir", "files", "bittorrent",
}

func (c *Aria2Client) AddURI(ctx context.Context, uris []string, options map[string]string) (string, error) {
	params := []any{uris}
	if options != nil {
		params = append(params, options)
	}
	result, err := c.Call(ctx, "aria2.addUri", params...)
	if err != nil {
		return "", err
	}
	var gid string
	return gid, json.Unmarshal(result, &gid)
}

func (c *Aria2Client) AddTorrent(ctx context.Context, torrent string, uris []string, options map[string]string) (string, error) {
	params := []any{torrent}
	if len(uris) > 0 {
		params = append(params, uris)
	}
	if options != nil {
		params = append(params, options)
	}
	result, err := c.Call(ctx, "aria2.addTorrent", params...)
	if err != nil {
		return "", err
	}
	var gid string
	return gid, json.Unmarshal(result, &gid)
}

func (c *Aria2Client) AddMetalink(ctx context.Context, metalink string, options map[string]string) ([]string, error) {
	params := []any{metalink}
	if options != nil {
		params = append(params, options)
	}
	result, err := c.Call(ctx, "aria2.addMetalink", params...)
	if err != nil {
		return nil, err
	}
	var gids []string
	return gids, json.Unmarshal(result, &gids)
}

func (c *Aria2Client) Remove(ctx context.Context, gid string) (string, error) {
	result, err := c.Call(ctx, "aria2.remove", gid)
	if err != nil {
		return "", err
	}
	var r string
	return r, json.Unmarshal(result, &r)
}

func (c *Aria2Client) ForceRemove(ctx context.Context, gid string) (string, error) {
	result, err := c.Call(ctx, "aria2.forceRemove", gid)
	if err != nil {
		return "", err
	}
	var r string
	return r, json.Unmarshal(result, &r)
}

func (c *Aria2Client) Pause(ctx context.Context, gid string) (string, error) {
	result, err := c.Call(ctx, "aria2.pause", gid)
	if err != nil {
		return "", err
	}
	var r string
	return r, json.Unmarshal(result, &r)
}

func (c *Aria2Client) Unpause(ctx context.Context, gid string) (string, error) {
	result, err := c.Call(ctx, "aria2.unpause", gid)
	if err != nil {
		return "", err
	}
	var r string
	return r, json.Unmarshal(result, &r)
}

func (c *Aria2Client) TellStatus(ctx context.Context, gid string, keys []string) (*DownloadStatus, error) {
	if len(keys) == 0 {
		keys = statusKeys
	}
	result, err := c.Call(ctx, "aria2.tellStatus", gid, keys)
	if err != nil {
		return nil, err
	}
	var status DownloadStatus
	return &status, json.Unmarshal(result, &status)
}

func (c *Aria2Client) TellActive(ctx context.Context, keys []string) ([]DownloadStatus, error) {
	if len(keys) == 0 {
		keys = statusKeys
	}
	result, err := c.Call(ctx, "aria2.tellActive", keys)
	if err != nil {
		return nil, err
	}
	var statuses []DownloadStatus
	return statuses, json.Unmarshal(result, &statuses)
}

func (c *Aria2Client) TellWaiting(ctx context.Context, offset int, num int, keys []string) ([]DownloadStatus, error) {
	if len(keys) == 0 {
		keys = statusKeys
	}
	result, err := c.Call(ctx, "aria2.tellWaiting", offset, num, keys)
	if err != nil {
		return nil, err
	}
	var statuses []DownloadStatus
	return statuses, json.Unmarshal(result, &statuses)
}

func (c *Aria2Client) TellStopped(ctx context.Context, offset int, num int, keys []string) ([]DownloadStatus, error) {
	if len(keys) == 0 {
		keys = statusKeys
	}
	result, err := c.Call(ctx, "aria2.tellStopped", offset, num, keys)
	if err != nil {
		return nil, err
	}
	var statuses []DownloadStatus
	return statuses, json.Unmarshal(result, &statuses)
}

func (c *Aria2Client) GetGlobalStat(ctx context.Context) (*GlobalStat, error) {
	result, err := c.Call(ctx, "aria2.getGlobalStat")
	if err != nil {
		return nil, err
	}
	var stat GlobalStat
	return &stat, json.Unmarshal(result, &stat)
}

func (c *Aria2Client) GetVersion(ctx context.Context) (*VersionInfo, error) {
	result, err := c.Call(ctx, "aria2.getVersion")
	if err != nil {
		return nil, err
	}
	var info VersionInfo
	return &info, json.Unmarshal(result, &info)
}

func (c *Aria2Client) RemoveDownloadResult(ctx context.Context, gid string) error {
	_, err := c.Call(ctx, "aria2.removeDownloadResult", gid)
	return err
}

func (c *Aria2Client) ChangeGlobalOption(ctx context.Context, options map[string]string) error {
	_, err := c.Call(ctx, "aria2.changeGlobalOption", options)
	return err
}

func (c *Aria2Client) Shutdown(ctx context.Context) error {
	_, err := c.Call(ctx, "aria2.shutdown")
	return err
}

func (c *Aria2Client) ForceShutdown(ctx context.Context) error {
	_, err := c.Call(ctx, "aria2.forceShutdown")
	return err
}

func (c *Aria2Client) PurgeDownloadResult(ctx context.Context) error {
	_, err := c.Call(ctx, "aria2.purgeDownloadResult")
	return err
}

func DownloadName(status *DownloadStatus) string {
	if status.Bittorrent != nil && status.Bittorrent.Info != nil && status.Bittorrent.Info.Name != "" {
		return status.Bittorrent.Info.Name
	}
	if len(status.Files) > 0 {
		if status.Files[0].Path != "" {
			return status.Files[0].Path
		}
		if len(status.Files[0].URIs) > 0 {
			return status.Files[0].URIs[0].URI
		}
	}
	return status.GID
}

func ParseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}