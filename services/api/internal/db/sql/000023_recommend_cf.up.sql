-- 推荐权重配置（单行表）
CREATE TABLE IF NOT EXISTS recommend_config (
    id                  SERIAL PRIMARY KEY,
    content_weight      REAL DEFAULT 0.4,
    cf_weight           REAL DEFAULT 0.4,
    popularity_weight   REAL DEFAULT 0.2,
    cf_min_cooccurrence INT DEFAULT 3,
    updated_at          TIMESTAMPTZ DEFAULT now()
);
INSERT INTO recommend_config (id) VALUES (1) ON CONFLICT DO NOTHING;

-- 协同过滤共现矩阵
CREATE TABLE IF NOT EXISTS cf_similarity (
    media_a_id  UUID NOT NULL,
    media_b_id  UUID NOT NULL,
    score       REAL NOT NULL,
    updated_at  TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY(media_a_id, media_b_id)
);
CREATE INDEX IF NOT EXISTS idx_cf_sim_a ON cf_similarity(media_a_id, score DESC);
CREATE INDEX IF NOT EXISTS idx_cf_sim_b ON cf_similarity(media_b_id, score DESC);
