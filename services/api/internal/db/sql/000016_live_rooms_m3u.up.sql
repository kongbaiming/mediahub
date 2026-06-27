-- M3U 导入元数据
ALTER TABLE live_rooms
    ADD COLUMN IF NOT EXISTS group_title VARCHAR(100),
    ADD COLUMN IF NOT EXISTS playlist_url TEXT;

CREATE INDEX IF NOT EXISTS idx_live_rooms_playlist_url ON live_rooms(playlist_url);
