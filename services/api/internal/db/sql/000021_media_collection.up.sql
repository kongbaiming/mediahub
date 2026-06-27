-- 000021_media_collection.up.sql
-- 为 media 表增加同系列（Collection）字段，支持 TMDB Collection 关联

ALTER TABLE media ADD COLUMN IF NOT EXISTS collection_id INT;
ALTER TABLE media ADD COLUMN IF NOT EXISTS collection_name VARCHAR(500);
ALTER TABLE media ADD COLUMN IF NOT EXISTS collection_poster_url TEXT;

-- collection_id 上的索引（用于查询同系列）
CREATE INDEX IF NOT EXISTS idx_media_collection_id ON media(collection_id) WHERE collection_id IS NOT NULL;