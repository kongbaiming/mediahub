package worker

import (
	"context"
	"math"
	"time"

	"github.com/mediahub/api/internal/domain/recommend"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshCFSimilarity 刷新协同过滤共现矩阵
//
// 算法：
// 1. 从 history 表构建用户-物品矩阵（profile_id → []media_id）
// 2. 计算 item-item 余弦相似度
// 3. 写入 cf_similarity 表
func RefreshCFSimilarity(ctx context.Context, db *gorm.DB, histRepo *repository.HistoryRepo) error {
	start := time.Now()
	logger.Info("开始刷新 CF 相似度矩阵")

	// 1. 拉取所有播放历史
	type historyRow struct {
		ProfileID uuid.UUID
		MediaID   uuid.UUID
		Completed bool
	}
	var rows []historyRow
	if err := db.WithContext(ctx).Table("history").
		Select("profile_id, media_id, completed").
		Where("progress > 0").
		Find(&rows).Error; err != nil {
		return err
	}

	if len(rows) == 0 {
		logger.Info("无播放历史，跳过 CF 计算")
		return nil
	}

	// 2. 构建 user-item 矩阵
	userItems := map[uuid.UUID]map[uuid.UUID]bool{}
	itemUsers := map[uuid.UUID]map[uuid.UUID]bool{}
	for _, r := range rows {
		if userItems[r.ProfileID] == nil {
			userItems[r.ProfileID] = map[uuid.UUID]bool{}
		}
		userItems[r.ProfileID][r.MediaID] = true
		if itemUsers[r.MediaID] == nil {
			itemUsers[r.MediaID] = map[uuid.UUID]bool{}
		}
		itemUsers[r.MediaID][r.ProfileID] = true
	}

	// 3. 计算 item-item 余弦相似度
	type simResult struct {
		A     uuid.UUID
		B     uuid.UUID
		Score float32
	}
	var similarities []simResult

	items := make([]uuid.UUID, 0, len(itemUsers))
	for id := range itemUsers {
		items = append(items, id)
	}

	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			a, b := items[i], items[j]
			usersA := itemUsers[a]
			usersB := itemUsers[b]

			// 余弦相似度 = |A ∩ B| / sqrt(|A| * |B|)
			intersection := 0
			for u := range usersA {
				if usersB[u] {
					intersection++
				}
			}
			if intersection == 0 {
				continue
			}

			score := float64(intersection) / math.Sqrt(float64(len(usersA)*len(usersB)))
			if score > 0.1 { // 阈值过滤
				similarities = append(similarities, simResult{A: a, B: b, Score: float32(score)})
				similarities = append(similarities, simResult{A: b, B: a, Score: float32(score)})
			}
		}
	}

	// 4. 写入数据库（事务：先清后插，避免崩溃导致数据丢失）
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&recommend.CFSimilarity{}).Error; err != nil {
			return err
		}

		if len(similarities) > 0 {
			batchSize := 500
			for i := 0; i < len(similarities); i += batchSize {
				end := i + batchSize
				if end > len(similarities) {
					end = len(similarities)
				}
				batch := similarities[i:end]
				if err := tx.CreateInBatches(batch, batchSize).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	logger.Info("CF 相似度矩阵刷新完成",
		"items", len(items),
		"pairs", len(similarities)/2,
		"duration", time.Since(start).String(),
	)
	return nil
}
