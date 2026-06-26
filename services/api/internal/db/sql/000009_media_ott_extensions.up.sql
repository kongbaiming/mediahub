-- 000009_media_ott_extensions.up.sql
-- OTT 呈现层 + 外部 Catalog ID（规划落地，v0.5 启用逻辑）

CREATE TABLE IF NOT EXISTS media_external_ids (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    provider        VARCHAR(32) NOT NULL,
    external_id     VARCHAR(64) NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    CONSTRAINT uq_media_external_provider UNIQUE (provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_media_external_media ON media_external_ids(media_id);

INSERT INTO media_external_ids (media_id, provider, external_id)
SELECT id, 'tmdb', tmdb_id::text
FROM media
WHERE tmdb_id IS NOT NULL
ON CONFLICT (provider, external_id) DO NOTHING;

INSERT INTO media_external_ids (media_id, provider, external_id)
SELECT id, 'douban', douban_id
FROM media
WHERE douban_id IS NOT NULL AND douban_id <> ''
ON CONFLICT (provider, external_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS media_renditions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id         UUID NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    protocol        VARCHAR(16) NOT NULL DEFAULT 'hls',
    profile         VARCHAR(32) NOT NULL,
    manifest_path   TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'building',
    error_message   TEXT,
    started_at      TIMESTAMP DEFAULT NOW(),
    ready_at        TIMESTAMP,
    expires_at      TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    CONSTRAINT uq_media_rendition UNIQUE (file_id, protocol, profile),
    CONSTRAINT chk_rendition_status CHECK (status IN ('building', 'ready', 'failed')),
    CONSTRAINT chk_rendition_protocol CHECK (protocol IN ('hls', 'dash'))
);

CREATE INDEX IF NOT EXISTS idx_media_renditions_file ON media_renditions(file_id);

CREATE INDEX IF NOT EXISTS idx_media_renditions_status ON media_renditions(status) WHERE status != 'ready';

ALTER TABLE media_files ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'primary';

UPDATE media_files SET role = 'primary' WHERE role IS NULL OR role = '';

ALTER TABLE media_files DROP CONSTRAINT IF EXISTS chk_media_files_role;

ALTER TABLE media_files ADD CONSTRAINT chk_media_files_role CHECK (role IN ('primary', 'alternate', 'extra'));
