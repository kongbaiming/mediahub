-- M3U 自动同步配置
CREATE TABLE IF NOT EXISTS live_m3u_sync_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    playlist_url TEXT NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    interval_minutes INT NOT NULL DEFAULT 1440,
    last_sync_at TIMESTAMP,
    last_sync_status VARCHAR(20),
    last_sync_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_live_m3u_sync_jobs_enabled ON live_m3u_sync_jobs(enabled);
