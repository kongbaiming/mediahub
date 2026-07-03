-- AB 测试事件统计（匿名）
CREATE TABLE IF NOT EXISTS ab_test_events (
    id              BIGSERIAL PRIMARY KEY,
    publication_id  UUID,
    variant         VARCHAR(1) NOT NULL,
    profile_id      UUID,
    event_type      VARCHAR(16) NOT NULL,
    row_key         VARCHAR(64),
    media_id        UUID,
    created_at      TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ab_events_pub ON ab_test_events(publication_id, variant, event_type);
CREATE INDEX IF NOT EXISTS idx_ab_events_date ON ab_test_events(created_at);

-- 用户自建片单
CREATE TABLE IF NOT EXISTS user_lists (
    id          BIGSERIAL PRIMARY KEY,
    profile_id  UUID NOT NULL,
    name        VARCHAR(128) NOT NULL,
    description TEXT,
    cover_url   TEXT,
    is_public   BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_user_lists_profile ON user_lists(profile_id);

CREATE TABLE IF NOT EXISTS user_list_items (
    list_id     BIGINT REFERENCES user_lists(id) ON DELETE CASCADE,
    media_id    UUID NOT NULL,
    sort_order  SMALLINT DEFAULT 0,
    added_at    TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY(list_id, media_id)
);
