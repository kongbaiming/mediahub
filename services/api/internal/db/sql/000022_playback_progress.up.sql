-- 跨设备续播进度表（独立于 history，支持 episode 级别精确续播）
CREATE TABLE IF NOT EXISTS playback_progress (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id  UUID NOT NULL,
    media_id    UUID NOT NULL,
    episode_id  UUID,
    position_sec INT NOT NULL DEFAULT 0,
    duration_sec INT NOT NULL DEFAULT 0,
    device      VARCHAR(50),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(profile_id, media_id, episode_id)
);
CREATE INDEX IF NOT EXISTS idx_playback_progress_profile ON playback_progress(profile_id, updated_at DESC);
