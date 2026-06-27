package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/live"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
)

// LiveConfig 直播服务配置
type LiveConfig struct {
	Enabled       bool
	RTMPHost      string // 对外 RTMP 地址（OBS 推流用）
	MediaMTXURL   string // 内部 HLS 源地址，如 http://mediamtx:8888
	MediaMTXAPIURL string // MediaMTX Control API，如 http://mediamtx:9997
	PublicAPIURL  string // 对外 API 基址（生成播放 URL），如 http://192.168.1.100:8081
}

// LiveService 直播间业务
type LiveService struct {
	repo   *repository.LiveRepo
	config LiveConfig
}

// NewLiveService 构造
func NewLiveService(repo *repository.LiveRepo, cfg LiveConfig) *LiveService {
	return &LiveService{repo: repo, config: cfg}
}

// CreateRoomRequest 创建直播间
type CreateRoomRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=200"`
	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`
	RoomType    string `json:"room_type"`  // push | iptv，默认 push
	SourceURL   string `json:"source_url"` // IPTV 拉流地址（m3u8）
}

// UpdateRoomRequest 更新直播间
type UpdateRoomRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	CoverURL    *string `json:"cover_url"`
	SourceURL   *string `json:"source_url"`
}

func (s *LiveService) Enabled() bool { return s.config.Enabled }

func (s *LiveService) Create(ctx context.Context, req CreateRoomRequest, userID uuid.UUID) (*live.RoomView, error) {
	if !s.config.Enabled {
		return nil, apperr.Validation("直播功能未启用")
	}
	roomType := live.RoomType(strings.TrimSpace(req.RoomType))
	if roomType == "" {
		roomType = live.RoomTypePush
	}
	if roomType != live.RoomTypePush && roomType != live.RoomTypeIPTV {
		return nil, apperr.Validation("无效的直播间类型")
	}

	key, err := generateStreamKey()
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "生成推流密钥失败")
	}

	now := time.Now()
	room := &live.Room{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		CoverURL:    strings.TrimSpace(req.CoverURL),
		RoomType:    roomType,
		Status:      live.StatusIdle,
		StreamKey:   key,
		CreatedBy:   &userID,
	}

	if roomType == live.RoomTypeIPTV {
		sourceURL, err := validateIPTVSourceURL(req.SourceURL)
		if err != nil {
			return nil, err
		}
		room.SourceURL = sourceURL
		room.Status = live.StatusLive
		room.StartedAt = &now
	}

	if err := s.repo.Create(ctx, room); err != nil {
		return nil, err
	}
	view := s.toView(room)
	return &view, nil
}

func (s *LiveService) Get(ctx context.Context, id string) (*live.RoomView, error) {
	room, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.syncRoomsWithMediaMTX(ctx, []live.Room{*room})
	s.syncIPTVRooms(ctx, []live.Room{*room})
	// 同步后重新读取
	room, err = s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	view := s.toView(room)
	return &view, nil
}

func (s *LiveService) List(ctx context.Context, status string, page, pageSize int) ([]live.RoomView, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	items, total, err := s.repo.List(ctx, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	s.syncRoomsWithMediaMTX(ctx, items)
	s.syncIPTVRooms(ctx, items)
	// 同步后重新拉列表
	items, total, err = s.repo.List(ctx, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	views := make([]live.RoomView, len(items))
	for i := range items {
		views[i] = s.toView(&items[i])
	}
	return views, total, nil
}

func (s *LiveService) Update(ctx context.Context, id string, req UpdateRoomRequest) (*live.RoomView, error) {
	room, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Title != nil {
		room.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		room.Description = strings.TrimSpace(*req.Description)
	}
	if req.CoverURL != nil {
		room.CoverURL = strings.TrimSpace(*req.CoverURL)
	}
	if req.SourceURL != nil && room.IsIPTV() {
		sourceURL, err := validateIPTVSourceURL(*req.SourceURL)
		if err != nil {
			return nil, err
		}
		room.SourceURL = sourceURL
	}
	if err := s.repo.Update(ctx, room); err != nil {
		return nil, err
	}
	view := s.toView(room)
	return &view, nil
}

func (s *LiveService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *LiveService) Stop(ctx context.Context, id string) (*live.RoomView, error) {
	room, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	room.Status = live.StatusEnded
	room.EndedAt = &now
	if err := s.repo.Update(ctx, room); err != nil {
		return nil, err
	}
	view := s.toView(room)
	return &view, nil
}

// OnPublish MediaMTX 推流开始回调
func (s *LiveService) OnPublish(ctx context.Context, streamPath string) error {
	key := extractStreamKey(streamPath)
	if key == "" {
		return apperr.Validation("无效的推流路径")
	}
	room, err := s.repo.GetByStreamKey(ctx, key)
	if err != nil {
		return err
	}
	if room.IsIPTV() {
		return nil
	}
	now := time.Now()
	room.Status = live.StatusLive
	room.StartedAt = &now
	room.EndedAt = nil
	return s.repo.Update(ctx, room)
}

// OnUnpublish MediaMTX 推流结束回调
func (s *LiveService) OnUnpublish(ctx context.Context, streamPath string) error {
	key := extractStreamKey(streamPath)
	if key == "" {
		return apperr.Validation("无效的推流路径")
	}
	room, err := s.repo.GetByStreamKey(ctx, key)
	if err != nil {
		return err
	}
	if room.IsIPTV() {
		return nil
	}
	if room.Status != live.StatusLive {
		return nil
	}
	now := time.Now()
	room.Status = live.StatusIdle
	room.EndedAt = &now
	return s.repo.Update(ctx, room)
}

// HLSPlaylistURL 返回 MediaMTX 上的主 HLS playlist 地址
func (s *LiveService) HLSPlaylistURL(streamKey string) string {
	return s.HLSMediaURL(streamKey, "index.m3u8", "")
}

// HLSMediaURL 返回 MediaMTX 上某个媒体资源的完整 URL（含 query）
func (s *LiveService) HLSMediaURL(streamKey, file, rawQuery string) string {
	base := strings.TrimRight(s.config.MediaMTXURL, "/")
	u := fmt.Sprintf("%s/%s/%s", base, streamKey, file)
	if rawQuery != "" {
		u += "?" + rawQuery
	}
	return u
}

func (s *LiveService) toView(room *live.Room) live.RoomView {
	view := live.RoomView{Room: *room}
	if room.IsIPTV() {
		view.PlayURL = fmt.Sprintf("/api/v1/live/rooms/%s/playlist.m3u8", room.ID)
		return view
	}
	view.StreamPath = room.StreamKey
	if s.config.RTMPHost != "" {
		view.RTMPURL = fmt.Sprintf("rtmp://%s/%s", strings.TrimPrefix(s.config.RTMPHost, "rtmp://"), room.StreamKey)
	}
	if s.config.MediaMTXURL != "" {
		view.HLSURL = s.HLSPlaylistURL(room.StreamKey)
	}
	view.PlayURL = fmt.Sprintf("/api/v1/live/rooms/%s/playlist.m3u8", room.ID)
	return view
}

func generateStreamKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func extractStreamKey(streamPath string) string {
	streamPath = strings.Trim(streamPath, "/")
	if streamPath == "" {
		return ""
	}
	parts := strings.Split(streamPath, "/")
	return parts[len(parts)-1]
}
