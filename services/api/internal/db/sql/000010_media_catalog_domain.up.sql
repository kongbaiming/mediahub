-- 000010_media_catalog_domain.up.sql
-- 媒资业务域：影人、演职员、分类、标签、专题专辑

CREATE TABLE IF NOT EXISTS persons (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                VARCHAR(300) NOT NULL,
    original_name       VARCHAR(300),
    tmdb_person_id      INT UNIQUE,
    profile_path        TEXT,
    biography           TEXT,
    birthday            DATE,
    place_of_birth      VARCHAR(200),
    gender              SMALLINT DEFAULT 0,
    known_for_department VARCHAR(50),
    popularity          DECIMAL(8,3) DEFAULT 0,
    created_at          TIMESTAMP DEFAULT NOW(),
    updated_at          TIMESTAMP DEFAULT NOW(),
    deleted_at          TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_persons_name ON persons USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_persons_tmdb ON persons(tmdb_person_id) WHERE tmdb_person_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS media_credits (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    person_id       UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    role            VARCHAR(32) NOT NULL,
    character_name  VARCHAR(300),
    billing_order   INT DEFAULT 0,
    created_at      TIMESTAMP DEFAULT NOW(),
    CONSTRAINT uq_media_credit UNIQUE (media_id, person_id, role, character_name)
);

CREATE INDEX IF NOT EXISTS idx_media_credits_media ON media_credits(media_id);

CREATE INDEX IF NOT EXISTS idx_media_credits_person ON media_credits(person_id);

CREATE TABLE IF NOT EXISTS categories (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id       UUID REFERENCES categories(id) ON DELETE SET NULL,
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(100) NOT NULL UNIQUE,
    kind            VARCHAR(20) NOT NULL DEFAULT 'genre',
    tmdb_genre_id   INT,
    sort_order      INT DEFAULT 0,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    CONSTRAINT chk_category_kind CHECK (kind IN ('genre', 'media_type', 'custom'))
);

CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);

CREATE TABLE IF NOT EXISTS media_categories (
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    category_id     UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    is_primary      BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (media_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_media_categories_cat ON media_categories(category_id);

CREATE TABLE IF NOT EXISTS tags (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    slug            VARCHAR(100) NOT NULL UNIQUE,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS media_tags (
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    tag_id          UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    source          VARCHAR(20) DEFAULT 'manual',
    PRIMARY KEY (media_id, tag_id),
    CONSTRAINT chk_tag_source CHECK (source IN ('tmdb', 'scanner', 'manual', 'user'))
);

CREATE INDEX IF NOT EXISTS idx_media_tags_tag ON media_tags(tag_id);

CREATE TABLE IF NOT EXISTS albums (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title           VARCHAR(500) NOT NULL,
    overview        TEXT,
    poster_url      TEXT,
    backdrop_url    TEXT,
    album_type      VARCHAR(32) DEFAULT 'collection',
    sort_order      INT DEFAULT 0,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    deleted_at      TIMESTAMP,
    CONSTRAINT chk_album_type CHECK (album_type IN ('collection', 'franchise', 'curated'))
);

CREATE TABLE IF NOT EXISTS album_items (
    album_id        UUID NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    media_id        UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    sort_order      INT DEFAULT 0,
    note            VARCHAR(200),
    PRIMARY KEY (album_id, media_id)
);

CREATE INDEX IF NOT EXISTS idx_album_items_media ON album_items(media_id);

INSERT INTO categories (name, slug, kind, sort_order) VALUES
    ('电影', 'movie', 'media_type', 1),
    ('剧集', 'tvshow', 'media_type', 2),
    ('动漫', 'anime', 'media_type', 3),
    ('纪录片', 'documentary', 'media_type', 4)
ON CONFLICT (slug) DO NOTHING;
