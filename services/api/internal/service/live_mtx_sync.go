package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mediahub/api/internal/domain/live"
	"github.com/mediahub/api/pkg/logger"
)

type mtxPathItem struct {
	Name      string `json:"name"`
	Online    bool   `json:"online"`
	Available bool   `json:"available"`
	Ready     bool   `json:"ready"`
}

type mtxPathListResp struct {
	Items []mtxPathItem `json:"items"`
}

// syncRoomsWithMediaMTX 从 MediaMTX Control API 同步推流状态（webhook 不可靠时的主路径）
func (s *LiveService) syncRoomsWithMediaMTX(ctx context.Context, rooms []live.Room) {
	if len(rooms) == 0 || s.config.MediaMTXAPIURL == "" {
		return
	}
	online, err := s.fetchMediaMTXOnlinePaths(ctx)
	if err != nil {
		logger.Debug("同步 MediaMTX 直播状态失败", "err", err)
		return
	}
	now := time.Now()
	for i := range rooms {
		room := &rooms[i]
		if room.IsIPTV() {
			continue
		}
		isOnline := online[room.StreamKey]
		if !isOnline {
			// 兼容旧版 API 字段
			for name, v := range online {
				if v && extractStreamKey(name) == room.StreamKey {
					isOnline = true
					break
				}
			}
		}
		if !s.applyOnlineStatus(ctx, room, isOnline, now) {
			continue
		}
	}
}

func (s *LiveService) applyOnlineStatus(ctx context.Context, room *live.Room, isOnline bool, now time.Time) bool {
	changed := false
	switch {
	case isOnline && room.Status != live.StatusLive:
		room.Status = live.StatusLive
		if room.StartedAt == nil {
			room.StartedAt = &now
		}
		room.EndedAt = nil
		changed = true
	case !isOnline && room.Status == live.StatusLive:
		room.Status = live.StatusIdle
		room.EndedAt = &now
		changed = true
	}
	if !changed {
		return false
	}
	if err := s.repo.Update(ctx, room); err != nil {
		logger.Warn("更新直播间状态失败", "room_id", room.ID, "err", err)
		return false
	}
	logger.Info("直播间状态已同步", "room_id", room.ID, "stream_key", room.StreamKey, "status", room.Status)
	return true
}

func (s *LiveService) fetchMediaMTXOnlinePaths(ctx context.Context) (map[string]bool, error) {
	base := strings.TrimRight(s.config.MediaMTXAPIURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v3/paths/list", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mediamtx api status %d", resp.StatusCode)
	}
	var data mtxPathListResp
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(data.Items))
	for _, item := range data.Items {
		out[item.Name] = item.Online || item.Available || item.Ready
	}
	return out, nil
}

// IsPathOnline 查询 MediaMTX 上指定 stream key 是否正在推流
func (s *LiveService) IsPathOnline(ctx context.Context, streamKey string) bool {
	if streamKey == "" || s.config.MediaMTXAPIURL == "" {
		return false
	}
	online, err := s.fetchMediaMTXOnlinePaths(ctx)
	if err != nil {
		return false
	}
	if online[streamKey] {
		return true
	}
	for name, v := range online {
		if v && extractStreamKey(name) == streamKey {
			return true
		}
	}
	return false
}

// GetRoomRaw 按 ID 读取直播间（不做 MediaMTX 同步，供 HLS 代理高频调用）
func (s *LiveService) GetRoomRaw(ctx context.Context, id string) (*live.Room, error) {
	return s.repo.GetByID(ctx, id)
}
