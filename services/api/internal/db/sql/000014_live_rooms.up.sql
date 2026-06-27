-- 直播间表
CREATE TABLE IF NOT EXISTS live_rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(200) NOT NULL,
    description TEXT,
    cover_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'idle',
    stream_key VARCHAR(64) NOT NULL UNIQUE,
    viewer_count INT NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_live_rooms_status ON live_rooms(status);
CREATE INDEX IF NOT EXISTS idx_live_rooms_stream_key ON live_rooms(stream_key);
CREATE INDEX IF NOT EXISTS idx_live_rooms_deleted_at ON live_rooms(deleted_at);
