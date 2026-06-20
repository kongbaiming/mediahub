-- 000004_layout_schema_fix.up.sql
-- 对齐 layouts / layout_publications 与 GORM 模型：
--   published_at  -> last_published_at
--   补 layout_publications.dynamic_rules

ALTER TABLE layouts ADD COLUMN IF NOT EXISTS last_published_at TIMESTAMP;

UPDATE layouts
SET last_published_at = published_at
WHERE last_published_at IS NULL AND published_at IS NOT NULL;

ALTER TABLE layouts DROP COLUMN IF EXISTS published_at;

ALTER TABLE layout_publications ADD COLUMN IF NOT EXISTS dynamic_rules JSONB;
