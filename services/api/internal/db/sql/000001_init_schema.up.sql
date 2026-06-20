-- ============================================================
-- MediaHub 数据库初始化
-- 文件：internal/db/sql/000001_init_schema.up.sql
-- 描述：核心表结构（媒资 / 布局 / 用户 / 历史）
-- ============================================================

-- ---------- 启用扩展 ----------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";   -- 模糊搜索

-- ---------- 媒资主表 ----------
CREATE TABLE IF NOT EXISTS media (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type            VARCHAR(20) NOT NULL,           -- movie | tvshow | anime | documentary
    title           VARCHAR(500) NOT NULL,
    original_title  VARCHAR(500),
    year            INT,
    runtime         INT,                             -- 分钟
    overview        TEXT,
    poster_url      TEXT,
    backdrop_url    TEXT,
    rating          DECIMAL(3,1) DEFAULT 0.0,       -- 0.0 ~ 10.0
    vote_count      INT DEFAULT 0,

    -- TMDB / 豆瓣 ID
    tmdb_id         INT,
    douban_id       VARCHAR(50),

    -- 标签（数组，Postgres 原生支持）
    genres          TEXT[] DEFAULT '{}',
    tags            TEXT[] DEFAULT '{}',

    -- 文件信息
    storage_path    TEXT NOT NULL,
    file_size       BIGINT DEFAULT 0,
    video_codec     VARCHAR(20),
    audio_codec     VARCHAR(20),
    resolution      VARCHAR(20),                     -- 1920x1080 等
    has_subtitle    BOOLEAN DEFAULT FALSE,
    container       VARCHAR(20),                     -- mkv / mp4
    is_adult        BOOLEAN DEFAULT FALSE,           -- 成人内容 (R18+)

    -- 状态
    scrape_status   VARCHAR(20) DEFAULT 'pending',   -- pending | scraping | done | failed
    scrape_error    TEXT,
    last_scrape_at  TIMESTAMP,

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),

    CONSTRAINT chk_media_type CHECK (type IN ('movie', 'tvshow', 'anime', 'documentary'))
);

CREATE INDEX idx_media_type        ON media(type);
CREATE INDEX idx_media_year        ON media(year);
CREATE INDEX idx_media_rating      ON media(rating DESC);
CREATE INDEX idx_media_created_at  ON media(created_at DESC);
CREATE INDEX idx_media_tmdb_id     ON media(tmdb_id);
CREATE INDEX idx_media_storage     ON media(storage_path);
CREATE INDEX idx_media_scrape      ON media(scrape_status) WHERE scrape_status != 'done';

-- 全文搜索（标题）
CREATE INDEX idx_media_title_trgm ON media USING gin (title gin_trgm_ops);

-- ---------- 季 ----------
CREATE TABLE IF NOT EXISTS seasons (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    season_number   INT NOT NULL,
    title           VARCHAR(500),
    overview        TEXT,
    poster_url      TEXT,
    air_date        DATE,
    episode_count   INT DEFAULT 0,
    created_at      TIMESTAMP DEFAULT NOW(),
    UNIQUE(media_id, season_number)
);

CREATE INDEX idx_seasons_media ON seasons(media_id);

-- ---------- 集 ----------
CREATE TABLE IF NOT EXISTS episodes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    season_id       UUID NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    episode_number  INT NOT NULL,
    title           VARCHAR(500),
    overview        TEXT,
    duration        INT,                             -- 秒
    file_path       TEXT,
    file_size       BIGINT,
    still_url       TEXT,                            -- 剧照
    created_at      TIMESTAMP DEFAULT NOW(),
    UNIQUE(season_id, episode_number)
);

CREATE INDEX idx_episodes_media ON episodes(media_id);
CREATE INDEX idx_episodes_season ON episodes(season_id);

-- ---------- 布局（核心）----------
CREATE TABLE IF NOT EXISTS layouts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id       UUID REFERENCES layouts(id) ON DELETE SET NULL,
    name            VARCHAR(200) NOT NULL,
    description     TEXT,
    is_template     BOOLEAN DEFAULT FALSE,
    version         INT DEFAULT 1,
    status          VARCHAR(20) DEFAULT 'draft',     -- draft | published | archived

    -- 布局 JSON（rows 数组）
    config          JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    published_at    TIMESTAMP
);

CREATE INDEX idx_layouts_parent   ON layouts(parent_id);
CREATE INDEX idx_layouts_status   ON layouts(status);
CREATE INDEX idx_layouts_template ON layouts(is_template) WHERE is_template = TRUE;

-- ---------- 布局发布（多端 + AB）----------
CREATE TABLE IF NOT EXISTS layout_publications (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    layout_id       UUID NOT NULL REFERENCES layouts(id) ON DELETE CASCADE,
    version         INT NOT NULL,
    target_platform VARCHAR(20) NOT NULL,           -- web | android-tv | tvos
    traffic_split   JSONB DEFAULT '{}'::jsonb,      -- AB 测试权重
    enabled         BOOLEAN DEFAULT TRUE,
    active_from     TIMESTAMP,
    active_to       TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),

    CONSTRAINT chk_publication_platform CHECK (target_platform IN ('web', 'android-tv', 'tvos'))
);

CREATE INDEX idx_publications_layout ON layout_publications(layout_id);
CREATE INDEX idx_publications_active ON layout_publications(target_platform, enabled) WHERE enabled = TRUE;

-- ---------- 用户 ----------
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username        VARCHAR(100) UNIQUE NOT NULL,
    password_hash   VARCHAR(200) NOT NULL,
    display_name    VARCHAR(200),
    email           VARCHAR(200),
    role            VARCHAR(20) DEFAULT 'member',    -- admin | member
    avatar_url      TEXT,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- ---------- Profile（家庭成员）----------
CREATE TABLE IF NOT EXISTS profiles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    avatar_url      TEXT,
    is_kid          BOOLEAN DEFAULT FALSE,
    pin_hash        VARCHAR(200),                    -- 家长锁
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_profiles_user ON profiles(user_id);

-- ---------- 收藏 ----------
CREATE TABLE IF NOT EXISTS favorites (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id      UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    favorite_type   VARCHAR(20) DEFAULT 'want',      -- want | watching | watched | liked
    rating          DECIMAL(3,1),
    created_at      TIMESTAMP DEFAULT NOW(),
    UNIQUE(profile_id, media_id)
);

CREATE INDEX idx_favorites_profile ON favorites(profile_id);
CREATE INDEX idx_favorites_media ON favorites(media_id);

-- ---------- 播放历史 ----------
CREATE TABLE IF NOT EXISTS history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id      UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    episode_id      UUID REFERENCES episodes(id) ON DELETE SET NULL,
    progress        INT DEFAULT 0,                   -- 秒
    duration        INT DEFAULT 0,
    completed       BOOLEAN DEFAULT FALSE,
    device          VARCHAR(50),                     -- web | android-tv | tvos
    updated_at      TIMESTAMP DEFAULT NOW(),
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_history_profile ON history(profile_id);
CREATE INDEX idx_history_media ON history(media_id);
CREATE INDEX idx_history_updated ON history(updated_at DESC);

-- ---------- 推荐缓存 ----------
CREATE TABLE IF NOT EXISTS recommendations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id      UUID REFERENCES profiles(id) ON DELETE CASCADE,
    algo            VARCHAR(50) NOT NULL,            -- hot | similar | cf | content
    media_ids       UUID[] NOT NULL,
    computed_at     TIMESTAMP DEFAULT NOW(),
    expires_at      TIMESTAMP NOT NULL
);

CREATE INDEX idx_recommendations_profile ON recommendations(profile_id, expires_at);

-- ---------- 任务审计 ----------
CREATE TABLE IF NOT EXISTS scrape_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id        UUID REFERENCES media(id) ON DELETE CASCADE,
    source          VARCHAR(50) NOT NULL,            -- tmdb | douban | manual
    action          VARCHAR(50) NOT NULL,            -- scrape | rescrape | fix
    status          VARCHAR(20) NOT NULL,            -- success | failed
    message         TEXT,
    duration_ms     INT,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_scrape_logs_media ON scrape_logs(media_id);
CREATE INDEX idx_scrape_logs_created ON scrape_logs(created_at DESC);

-- ============================================================
-- 完成
-- ============================================================
