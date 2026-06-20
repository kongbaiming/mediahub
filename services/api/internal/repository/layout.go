package repository

import (
	"context"
	"errors"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/layout"

	"gorm.io/gorm"
)

// LayoutRepo 布局仓储
type LayoutRepo struct {
	db *gorm.DB
}

// NewLayoutRepo 构造布局仓储
func NewLayoutRepo(db *gorm.DB) *LayoutRepo {
	return &LayoutRepo{db: db}
}

// Create 创建布局
func (r *LayoutRepo) Create(ctx context.Context, l *layout.Layout) error {
	if err := r.db.WithContext(ctx).Create(l).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "创建布局失败")
	}
	return nil
}

// GetByID 按 ID 获取（含 Publications）
func (r *LayoutRepo) GetByID(ctx context.Context, id string) (*layout.Layout, error) {
	var l layout.Layout
	if err := r.db.WithContext(ctx).
		Preload("Publications").
		First(&l, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("布局不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询布局失败")
	}
	return &l, nil
}

// List 列表查询
func (r *LayoutRepo) List(ctx context.Context, isTemplate *bool, status string) ([]layout.Layout, error) {
	q := r.db.WithContext(ctx).Model(&layout.Layout{})
	if isTemplate != nil {
		q = q.Where("is_template = ?", *isTemplate)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var items []layout.Layout
	if err := q.Order("updated_at DESC").Find(&items).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询布局列表失败")
	}
	return items, nil
}

// Update 更新布局
func (r *LayoutRepo) Update(ctx context.Context, l *layout.Layout) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(l).Updates(map[string]any{
			"name":        l.Name,
			"description": l.Description,
			"config":      l.Config,
			"is_template": l.IsTemplate,
			"parent_id":   l.ParentID,
			"version":     l.Version,
			"status":      l.Status,
		}).Error; err != nil {
			return apperr.Wrap(err, apperr.CodeInternal, "更新布局失败")
		}
		return nil
	})
}

// Delete 删除布局
func (r *LayoutRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&layout.Layout{}, "id = ?", id).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "删除布局失败")
	}
	return nil
}

// GetActivePublications 拉取指定平台当前活跃的所有发布（AB 用）
func (r *LayoutRepo) GetActivePublications(ctx context.Context, platform string, now time.Time) ([]layout.Publication, error) {
	var pubs []layout.Publication
	err := r.db.WithContext(ctx).
		Where("target_platform = ? AND enabled = ?", platform, true).
		Where("active_from IS NULL OR active_from <= ?", now).
		Where("active_to IS NULL OR active_to >= ?", now).
		Order("created_at DESC").
		Find(&pubs).Error
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询发布失败")
	}
	return pubs, nil
}

// GetPublishedForPlatform 拉取指定平台的已发布布局（AB 选取）
// userHash 用于 AB 分桶（一般为 profile_id）
func (r *LayoutRepo) GetPublishedForPlatform(ctx context.Context, platform, userHash string) (*layout.Layout, error) {
	now := time.Now()

	// 1. 拉取所有活跃发布
	pubs, err := r.GetActivePublications(ctx, platform, now)
	if err != nil {
		return nil, err
	}
	if len(pubs) == 0 {
		return nil, apperr.NotFound("该平台暂无已发布布局")
	}

	// 2. 应用动态规则过滤
	filtered := make([]layout.Publication, 0, len(pubs))
	for _, p := range pubs {
		if p.DynamicRules == nil || p.DynamicRules.Matches(now) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return nil, apperr.NotFound("当前时段无匹配布局")
	}

	// 3. AB 分桶
	chosen := pickByTrafficSplit(filtered, userHash)
	if chosen == nil {
		return nil, apperr.NotFound("AB 分桶失败")
	}

	// 4. 加载布局
	var l layout.Layout
	if err := r.db.WithContext(ctx).First(&l, "id = ?", chosen.LayoutID).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询布局失败")
	}
	return &l, nil
}

// CreatePublication 创建发布记录
func (r *LayoutRepo) CreatePublication(ctx context.Context, p *layout.Publication) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "创建发布记录失败")
	}
	return nil
}

// ListPublications 列出布局的所有发布
func (r *LayoutRepo) ListPublications(ctx context.Context, layoutID string) ([]layout.Publication, error) {
	var pubs []layout.Publication
	if err := r.db.WithContext(ctx).
		Where("layout_id = ?", layoutID).
		Order("created_at DESC").
		Find(&pubs).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询发布失败")
	}
	return pubs, nil
}

// DisablePublication 禁用发布
func (r *LayoutRepo) DisablePublication(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Model(&layout.Publication{}).
		Where("id = ?", id).
		Update("enabled", false).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "禁用发布失败")
	}
	return nil
}

// ---- AB 分桶辅助 ----

// pickByTrafficSplit 根据 traffic_split 权重选取一项
// userHash 为空时取第一项
func pickByTrafficSplit(pubs []layout.Publication, userHash string) *layout.Publication {
	if len(pubs) == 1 {
		return &pubs[0]
	}

	// 计算总权重
	total := 0
	weights := make(map[int]int, len(pubs)) // idx -> weight
	for i, p := range pubs {
		if p.TrafficSplit == nil || len(p.TrafficSplit) == 0 {
			// 没设权重 → 平均
			weights[i] = 100 / len(pubs)
		} else {
			// 取第一个 key（AB label）的权重
			for _, w := range p.TrafficSplit {
				weights[i] = w
				break
			}
		}
		total += weights[i]
	}
	if total == 0 {
		return &pubs[0]
	}

	// 计算 userHash 的桶值（0~total）
	var bucket int
	if userHash != "" {
		bucket = hashToBucket(userHash, total)
	} else {
		bucket = 0
	}

	// 累加找区间
	cum := 0
	for i := range pubs {
		cum += weights[i]
		if bucket < cum {
			return &pubs[i]
		}
	}
	return &pubs[len(pubs)-1]
}

// hashToBucket 把字符串哈希到 [0, total) 区间
func hashToBucket(s string, total int) int {
	if total <= 0 {
		return 0
	}
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return int(h % uint32(total))
}
