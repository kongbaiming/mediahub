-- 直播间支持 IPTV 拉流
ALTER TABLE live_rooms
    ADD COLUMN IF NOT EXISTS room_type VARCHAR(20) NOT NULL DEFAULT 'push',
    ADD COLUMN IF NOT EXISTS source_url TEXT;

CREATE INDEX IF NOT EXISTS idx_live_rooms_room_type ON live_rooms(room_type);
