-- 000002_add_deleted_at.up.sql
-- 给所有 embed 了 common.BaseModel 的表加 deleted_at 列 + 索引，
-- 配合 GORM soft-delete（gorm.DeletedAt）。
--
-- 用 IF NOT EXISTS 保证幂等：第一次跑加列，后续跳过。
-- 涉及表：users / profiles / media / seasons / episodes /
--        history / favorites / recommendations /
--        layouts / layout_publications

ALTER TABLE users               ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE profiles            ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE media               ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE seasons             ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE episodes            ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE history             ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE favorites           ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE recommendations     ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE layouts             ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE layout_publications ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_users_deleted_at               ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_profiles_deleted_at            ON profiles(deleted_at);
CREATE INDEX IF NOT EXISTS idx_media_deleted_at               ON media(deleted_at);
CREATE INDEX IF NOT EXISTS idx_seasons_deleted_at             ON seasons(deleted_at);
CREATE INDEX IF NOT EXISTS idx_episodes_deleted_at            ON episodes(deleted_at);
CREATE INDEX IF NOT EXISTS idx_history_deleted_at             ON history(deleted_at);
CREATE INDEX IF NOT EXISTS idx_favorites_deleted_at           ON favorites(deleted_at);
CREATE INDEX IF NOT EXISTS idx_recommendations_deleted_at     ON recommendations(deleted_at);
CREATE INDEX IF NOT EXISTS idx_layouts_deleted_at             ON layouts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_layout_publications_deleted_at ON layout_publications(deleted_at);