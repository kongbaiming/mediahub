-- 000011_ott_gaps.up.sql
-- OTT 补全：可播状态、分级、字幕、预告、海报变体、作品关系、Profile 分龄

ALTER TABLE media ADD COLUMN IF NOT EXISTS availability_status VARCHAR(20) DEFAULT 'processing'
ALTER TABLE media ADD COLUMN IF NOT EXISTS available_at TIMESTAMP

CREATE INDEX IF NOT EXISTS idx_media_availability ON media(availability_status)

CREATE TABLE IF NOT EXISTS content_ratings (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    country         VARCHAR(8) NOT NULL DEFAULT 'US',
    system          VARCHAR(32) NOT NULL DEFAULT 'tmdb',
    rating          VARCHAR(32) NOT NULL,
    advisories      TEXT[] DEFAULT '{}',
    created_at      TIMESTAMP DEFAULT NOW(),
    CONSTRAINT uq_content_rating UNIQUE (media_id, country, system)
)

CREATE INDEX IF NOT EXISTS idx_content_ratings_media ON content_ratings(media_id)

CREATE TABLE IF NOT EXISTS subtitle_tracks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_file_id   UUID REFERENCES media_files(id) ON DELETE CASCADE,
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    episode_id      UUID REFERENCES episodes(id) ON DELETE CASCADE,
    language        VARCHAR(16) NOT NULL DEFAULT 'zh',
    format          VARCHAR(16) NOT NULL DEFAULT 'srt',
    path            TEXT,
    label           VARCHAR(100),
    source          VARCHAR(20) DEFAULT 'manual',
    is_default      BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMP DEFAULT NOW(),
    CONSTRAINT chk_subtitle_source CHECK (source IN ('embedded', 'subhd', 'manual', 'opensubtitles'))
)

CREATE INDEX IF NOT EXISTS idx_subtitle_tracks_media ON subtitle_tracks(media_id)

CREATE TABLE IF NOT EXISTS media_extras (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    extra_type      VARCHAR(32) NOT NULL DEFAULT 'trailer',
    title           VARCHAR(300),
    source          VARCHAR(20) DEFAULT 'tmdb',
    file_path       TEXT,
    external_url    TEXT,
    external_key    VARCHAR(64),
    duration_sec    INT DEFAULT 0,
    created_at      TIMESTAMP DEFAULT NOW(),
    CONSTRAINT chk_extra_type CHECK (extra_type IN ('trailer', 'teaser', 'behind_the_scenes', 'clip')),
    CONSTRAINT uq_media_extra_key UNIQUE (media_id, extra_type, external_key)
)

CREATE INDEX IF NOT EXISTS idx_media_extras_media ON media_extras(media_id)

CREATE TABLE IF NOT EXISTS media_artworks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    season_id       UUID REFERENCES seasons(id) ON DELETE CASCADE,
    episode_id      UUID REFERENCES episodes(id) ON DELETE CASCADE,
    art_type        VARCHAR(32) NOT NULL,
    locale          VARCHAR(16) DEFAULT '',
    url             TEXT NOT NULL,
    width           INT DEFAULT 0,
    height          INT DEFAULT 0,
    created_at      TIMESTAMP DEFAULT NOW(),
    CONSTRAINT chk_art_type CHECK (art_type IN ('poster', 'backdrop', 'logo', 'still'))
)

CREATE INDEX IF NOT EXISTS idx_media_artworks_media ON media_artworks(media_id)

CREATE TABLE IF NOT EXISTS media_relations (
    from_media_id   UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    to_media_id     UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    relation_type   VARCHAR(32) NOT NULL,
    sort_order      INT DEFAULT 0,
    PRIMARY KEY (from_media_id, to_media_id, relation_type),
    CONSTRAINT chk_relation_type CHECK (relation_type IN ('sequel', 'prequel', 'same_franchise', 'remake', 'similar'))
)

CREATE TABLE IF NOT EXISTS profile_content_policy (
    profile_id          UUID PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
    max_rating_level    INT DEFAULT 100,
    block_adult         BOOLEAN DEFAULT FALSE,
    updated_at          TIMESTAMP DEFAULT NOW()
)

UPDATE media SET availability_status = 'processing' WHERE availability_status IS NULL
