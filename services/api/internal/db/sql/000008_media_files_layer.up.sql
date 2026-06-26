-- 000008_media_files_layer.up.sql
-- 媒资架构 v2：作品层(media) + 结构层(seasons/episodes) + 文件层(media_files)

CREATE TABLE IF NOT EXISTS media_files (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    episode_id      UUID REFERENCES episodes(id) ON DELETE CASCADE,
    path            TEXT NOT NULL,
    file_size       BIGINT DEFAULT 0,
    duration_sec    INT DEFAULT 0,
    video_codec     VARCHAR(32),
    audio_codec     VARCHAR(32),
    width           INT DEFAULT 0,
    height          INT DEFAULT 0,
    resolution      VARCHAR(20),
    container       VARCHAR(20),
    has_subtitle    BOOLEAN DEFAULT FALSE,
    is_primary      BOOLEAN DEFAULT TRUE,
    probe_status    VARCHAR(20) DEFAULT 'pending',
    probed_at       TIMESTAMP,
    source          VARCHAR(20) DEFAULT 'scan',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    deleted_at      TIMESTAMP,
    CONSTRAINT uq_media_files_path UNIQUE (path),
    CONSTRAINT chk_media_files_probe CHECK (probe_status IN ('pending', 'done', 'failed'))
)

CREATE INDEX IF NOT EXISTS idx_media_files_media ON media_files(media_id)

CREATE INDEX IF NOT EXISTS idx_media_files_episode ON media_files(episode_id) WHERE episode_id IS NOT NULL

CREATE INDEX IF NOT EXISTS idx_media_files_probe ON media_files(probe_status) WHERE probe_status != 'done'

ALTER TABLE media ADD COLUMN IF NOT EXISTS kind VARCHAR(20)

UPDATE media SET kind = 'series' WHERE type IN ('tvshow', 'anime') AND (kind IS NULL OR kind = '')

UPDATE media SET kind = 'single' WHERE kind IS NULL OR kind = ''

ALTER TABLE media DROP CONSTRAINT IF EXISTS chk_media_kind

ALTER TABLE media ADD CONSTRAINT chk_media_kind CHECK (kind IN ('single', 'series'))

INSERT INTO media_files (
    media_id, episode_id, path, file_size, video_codec, audio_codec,
    resolution, container, has_subtitle, is_primary, probe_status, source, probed_at
)
SELECT
    m.id, NULL, m.storage_path, COALESCE(m.file_size, 0),
    m.video_codec, m.audio_codec, m.resolution, m.container,
    COALESCE(m.has_subtitle, FALSE), TRUE,
    CASE WHEN m.video_codec IS NOT NULL AND m.video_codec <> '' THEN 'done' ELSE 'pending' END,
    'scan', m.updated_at
FROM media m
WHERE m.kind = 'single'
  AND m.storage_path IS NOT NULL
  AND m.storage_path <> ''
ON CONFLICT (path) DO NOTHING

INSERT INTO media_files (media_id, episode_id, path, file_size, is_primary, probe_status, source)
SELECT
    e.media_id, e.id, e.file_path, COALESCE(e.file_size, 0), TRUE, 'pending', 'scan'
FROM episodes e
WHERE e.file_path IS NOT NULL
  AND e.file_path <> ''
ON CONFLICT (path) DO NOTHING
