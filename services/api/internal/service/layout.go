package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/layout"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
)

// LayoutService 布局业务
type LayoutService struct {
	repo            *repository.LayoutRepo
	feedInvalidator func(ctx context.Context, platform string) error // Feed 缓存失效回调
}

// NewLayoutService 构造
func NewLayoutService(repo *repository.LayoutRepo) *LayoutService {
	return &LayoutService{repo: repo}
}

// CreateRequest 创建请求
type CreateRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	IsTemplate  bool                    `json:"is_template"`
	ParentID    *string                 `json:"parent_id,omitempty"`
	Config      layout.LayoutConfig     `json:"config"`
}

// UpdateRequest 更新请求
type UpdateRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	IsTemplate  *bool                  `json:"is_template,omitempty"`
	Config      *layout.LayoutConfig   `json:"config,omitempty"`
	ParentID    *string                `json:"parent_id,omitempty"`
	ParentIDSet bool                   `json:"-"` // handler 解析 body 时设置
}

// PublishRequest 发布请求
type PublishRequest struct {
	TargetPlatform common.Platform           `json:"target_platform" binding:"required"`
	TrafficSplit   map[string]int            `json:"traffic_split,omitempty"`
	ActiveFrom     *time.Time                `json:"active_from,omitempty"`
	ActiveTo       *time.Time                `json:"active_to,omitempty"`
	DynamicRules   *layout.DynamicRules      `json:"dynamic_rules,omitempty"`
}

// Create 创建布局
func (s *LayoutService) Create(ctx context.Context, req CreateRequest) (*layout.Layout, error) {
	if req.Name == "" {
		return nil, apperr.Validation(map[string]string{"name": "名称不能为空"})
	}
	l := &layout.Layout{
		Name:        req.Name,
		Description: req.Description,
		IsTemplate:  req.IsTemplate,
		Status:      common.LayoutDraft,
		Config:      req.Config,
	}
	if l.Config.Rows == nil {
		l.Config.Rows = []layout.Row{}
	}
	if req.ParentID != nil {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, apperr.Validation(map[string]string{"parent_id": "格式错误"})
		}
		l.ParentID = &pid
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// Get 获取
func (s *LayoutService) Get(ctx context.Context, id string) (*layout.Layout, error) {
	return s.repo.GetByID(ctx, id)
}

// GetForEditor 编辑器加载：合并父布局行并标记 _inherited
func (s *LayoutService) GetForEditor(ctx context.Context, id string) (*layout.Layout, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	childRowIDs := make(map[string]bool, len(l.Config.Rows))
	for _, r := range l.Config.Rows {
		childRowIDs[r.ID] = true
	}

	merged, err := s.resolveInheritance(ctx, l)
	if err != nil {
		return nil, err
	}

	for i := range merged.Config.Rows {
		if !childRowIDs[merged.Config.Rows[i].ID] {
			merged.Config.Rows[i].Inherited = true
		}
	}
	return merged, nil
}

// List 列表
func (s *LayoutService) List(ctx context.Context, isTemplate *bool, status string) ([]layout.Layout, error) {
	return s.repo.List(ctx, isTemplate, status)
}

// Update 更新
func (s *LayoutService) Update(ctx context.Context, id string, req UpdateRequest) (*layout.Layout, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		l.Name = *req.Name
	}
	if req.Description != nil {
		l.Description = *req.Description
	}
	if req.IsTemplate != nil {
		l.IsTemplate = *req.IsTemplate
	}
	if req.ParentIDSet {
		if req.ParentID == nil || *req.ParentID == "" {
			l.ParentID = nil
		} else {
			pid, err := uuid.Parse(*req.ParentID)
			if err != nil {
				return nil, apperr.Validation(map[string]string{"parent_id": "格式错误"})
			}
			l.ParentID = &pid
		}
	}
	if req.Config != nil {
		// 去掉编辑器标记字段，避免写入 DB
		cfg := *req.Config
		for i := range cfg.Rows {
			cfg.Rows[i].Inherited = false
		}
		l.Config = cfg
		l.Version++
	}
	if err := s.repo.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// Delete 删除布局
func (s *LayoutService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Publish 发布
func (s *LayoutService) Publish(ctx context.Context, id string, req PublishRequest) (*layout.Layout, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 校验 traffic_split
	if len(req.TrafficSplit) > 0 {
		total := 0
		for _, w := range req.TrafficSplit {
			if w < 0 {
				return nil, apperr.Validation(map[string]string{
					"traffic_split": "权重不能为负",
				})
			}
			total += w
		}
		if total == 0 {
			return nil, apperr.Validation(map[string]string{
				"traffic_split": "权重总和不能为 0",
			})
		}
	}

	// 校验 dynamic_rules
	if req.DynamicRules != nil {
		if req.DynamicRules.HourOfDay != nil {
			h := req.DynamicRules.HourOfDay
			if h.Start < 0 || h.Start > 23 || h.End < 0 || h.End > 23 {
				return nil, apperr.Validation(map[string]string{
					"dynamic_rules.hour_of_day": "小时必须在 0-23",
				})
			}
		}
	}

	pub := &layout.Publication{
		LayoutID:       l.ID,
		Version:        l.Version,
		TargetPlatform: req.TargetPlatform,
		TrafficSplit:   layout.TrafficSplit(req.TrafficSplit),
		DynamicRules:   req.DynamicRules,
		ActiveFrom:     req.ActiveFrom,
		ActiveTo:       req.ActiveTo,
		Enabled:        true,
	}
	if err := s.repo.CreatePublication(ctx, pub); err != nil {
		return nil, err
	}

	l.Status = common.LayoutPublished
	now := time.Now()
	l.LastPublishedAt = &now
	if err := s.repo.Update(ctx, l); err != nil {
		return nil, err
	}

	// 失效 Feed 缓存（让客户端下次拉取能看到新布局）
	if s.feedInvalidator != nil {
		_ = s.feedInvalidator(ctx, string(req.TargetPlatform))
	}

	return l, nil
}

// SetFeedInvalidator 注入 Feed 失效回调（由 main.go 在装配时注入）
func (s *LayoutService) SetFeedInvalidator(fn func(ctx context.Context, platform string) error) {
	s.feedInvalidator = fn
}

// GetFeed 播放端拉取 Feed（含模板继承）
func (s *LayoutService) GetFeed(ctx context.Context, platform string, profileID string) (*layout.Feed, error) {
	l, err := s.repo.GetPublishedForPlatform(ctx, platform, profileID)
	if err != nil {
		return nil, err
	}

	// 模板继承（递归合并所有祖先）
	merged, err := s.resolveInheritance(ctx, l)
	if err != nil {
		return nil, err
	}

	return &layout.Feed{
		Version:   merged.Version,
		Platform:  platform,
		UpdatedAt: merged.UpdatedAt,
		Rows:      []layout.FeedRow{}, // 数据由 FeedService 填充
	}, nil
}

// resolveInheritance 解析模板继承（递归）
func (s *LayoutService) resolveInheritance(ctx context.Context, l *layout.Layout) (*layout.Layout, error) {
	if l.ParentID == nil {
		return l, nil
	}

	// 防止循环引用
	visited := map[uuid.UUID]bool{l.ID: true}
	current := l
	for current.ParentID != nil {
		if visited[*current.ParentID] {
			return nil, errors.New("布局继承存在循环引用")
		}
		visited[*current.ParentID] = true

		parent, err := s.repo.GetByID(ctx, current.ParentID.String())
		if err != nil {
			return nil, fmt.Errorf("加载父布局: %w", err)
		}
		// 合并 parent 和 current（child 覆盖 parent）
		current = mergeLayoutInherit(parent, current)

		// 防止无限深
		if len(visited) > 10 {
			return nil, errors.New("布局继承深度超过 10 层")
		}
	}
	return current, nil
}

// mergeLayoutInherit 把 child 合并到 parent 上（child 字段优先）
func mergeLayoutInherit(parent, child *layout.Layout) *layout.Layout {
	merged := *parent // copy

	// rows：按 id 去重，child 中的 row 覆盖 parent 中的 row
	mergedRowsMap := make(map[string]layout.Row, len(parent.Config.Rows)+len(child.Config.Rows))
	rowOrder := make([]string, 0, len(parent.Config.Rows)+len(child.Config.Rows))

	for _, r := range parent.Config.Rows {
		mergedRowsMap[r.ID] = r
		rowOrder = append(rowOrder, r.ID)
	}
	for _, r := range child.Config.Rows {
		if _, exists := mergedRowsMap[r.ID]; !exists {
			rowOrder = append(rowOrder, r.ID)
		}
		mergedRowsMap[r.ID] = r // child 覆盖
	}

	merged.Config.Rows = make([]layout.Row, 0, len(rowOrder))
	for _, id := range rowOrder {
		merged.Config.Rows = append(merged.Config.Rows, mergedRowsMap[id])
	}

	// 主题也以 child 为准
	if child.Config.Theme != "" {
		merged.Config.Theme = child.Config.Theme
	}

	return &merged
}

// ListPublications 列出布局的所有发布
func (s *LayoutService) ListPublications(ctx context.Context, layoutID string) ([]layout.Publication, error) {
	return s.repo.ListPublications(ctx, layoutID)
}

// DisablePublication 禁用某个发布
func (s *LayoutService) DisablePublication(ctx context.Context, id string) error {
	return s.repo.DisablePublication(ctx, id)
}

// uuidParse 辅助
func uuidParse(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
