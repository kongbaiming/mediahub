-- 媒资库自动扫描配置（单行）
CREATE TABLE IF NOT EXISTS media_scan_config (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    interval_minutes INT NOT NULL DEFAULT 30,
    last_scan_at TIMESTAMP,
    last_scan_status VARCHAR(20),
    last_scan_message TEXT,
    last_scan_added INT NOT NULL DEFAULT 0,
    last_scan_total INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO media_scan_config (id, enabled, interval_minutes)
VALUES (1, TRUE, 30)
ON CONFLICT (id) DO NOTHING;
