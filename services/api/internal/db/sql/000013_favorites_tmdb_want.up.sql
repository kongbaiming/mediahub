-- 支持库外（TMDB）想看：media_id 可空，增加 tmdb 快照字段

ALTER TABLE favorites ALTER COLUMN media_id DROP NOT NULL;

ALTER TABLE favorites ADD COLUMN IF NOT EXISTS tmdb_id INT;
ALTER TABLE favorites ADD COLUMN IF NOT EXISTS media_type VARCHAR(20);
ALTER TABLE favorites ADD COLUMN IF NOT EXISTS title VARCHAR(500);
ALTER TABLE favorites ADD COLUMN IF NOT EXISTS year INT;
ALTER TABLE favorites ADD COLUMN IF NOT EXISTS poster_url TEXT;

ALTER TABLE favorites DROP CONSTRAINT IF EXISTS favorites_profile_id_media_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_favorites_profile_media
  ON favorites(profile_id, media_id)
  WHERE media_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_favorites_profile_tmdb_want
  ON favorites(profile_id, tmdb_id, favorite_type)
  WHERE tmdb_id IS NOT NULL AND media_id IS NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_favorites_tmdb ON favorites(tmdb_id) WHERE tmdb_id IS NOT NULL;
