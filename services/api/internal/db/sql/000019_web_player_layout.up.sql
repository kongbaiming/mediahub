-- 000019_web_player_layout.up.sql
-- 重新编排 Web 播放端首页布局（schema: web-v2）

UPDATE layouts
SET
  name = CASE WHEN name = '' OR name IS NULL THEN 'Web 播放端首页' ELSE name END,
  description = COALESCE(NULLIF(description, ''), '播放端首页：Hero · 继续观看 · 分区浏览 · 智能推荐'),
  config = '{
    "theme": "dark",
    "global": { "layout_schema": "web-v2" },
    "rows": [
      {
        "id": "hero-web",
        "type": "hero-banner",
        "title": "精选推荐",
        "card_style": "banner",
        "visible": true,
        "source": { "type": "recommend-algorithm", "params": { "algo": "hot", "limit": 8 } }
      },
      {
        "id": "continue-web",
        "type": "shelf",
        "title": "继续观看",
        "subtitle": "从上次停下的地方继续",
        "card_style": "landscape",
        "visible": true,
        "source": { "type": "continue-watching", "params": { "limit": 12 } }
      },
      {
        "id": "banner-live",
        "type": "text-banner",
        "title": "📡 电视直播",
        "subtitle": "浏览 IPTV 频道与推流直播，支持按栏目筛选",
        "visible": true,
        "config": { "action": { "type": "live", "label": "进入直播" } },
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "divider-recent",
        "type": "divider",
        "title": "最新更新",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "recent-web",
        "type": "shelf",
        "title": "最近入库",
        "subtitle": "刚刚加入媒体库的内容",
        "card_style": "poster",
        "visible": true,
        "source": { "type": "recently-added", "params": { "limit": 20 } }
      },
      {
        "id": "divider-hot",
        "type": "divider",
        "title": "热门推荐",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "hot-web",
        "type": "topic",
        "title": "高分影视",
        "subtitle": "库内评分最高的内容",
        "card_style": "landscape",
        "visible": true,
        "source": { "type": "trending", "params": { "limit": 16 } }
      },
      {
        "id": "divider-movies",
        "type": "divider",
        "title": "电影",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "movies-grid",
        "type": "category-grid",
        "title": "电影库",
        "card_style": "poster",
        "visible": true,
        "source": { "type": "library", "params": { "type": "movie", "sort": "rating", "limit": 24 } }
      },
      {
        "id": "divider-tv",
        "type": "divider",
        "title": "剧集",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "tv-grid",
        "type": "category-grid",
        "title": "剧集库",
        "card_style": "landscape",
        "visible": true,
        "source": { "type": "library", "params": { "type": "tvshow", "sort": "rating", "limit": 24 } }
      },
      {
        "id": "divider-guess",
        "type": "divider",
        "title": "发现更多",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "guess-you-like-web",
        "type": "shelf",
        "title": "猜你喜欢",
        "subtitle": "根据观影偏好智能推荐",
        "card_style": "poster",
        "visible": true,
        "source": { "type": "guess-you-like", "params": { "limit": 16, "discover_limit": 4 } }
      }
    ]
  }'::jsonb,
  version = version + 1,
  status = 'published',
  last_published_at = NOW(),
  updated_at = NOW()
WHERE (config->'global'->>'layout_schema' IS DISTINCT FROM 'web-v2')
  AND (
    id = 'd7cedd8e-5598-4f9b-bc82-6dc1a954b362'::uuid
    OR id IN (
      SELECT layout_id FROM layout_publications
      WHERE target_platform = 'web' AND enabled = TRUE
    )
    OR EXISTS (
      SELECT 1 FROM jsonb_array_elements(COALESCE(config->'rows', '[]'::jsonb)) e
      WHERE e->>'id' = 'hero-web'
    )
  );

-- 若无 Web 布局则创建
INSERT INTO layouts (id, name, description, status, config, version, last_published_at, created_at, updated_at)
SELECT
  'd7cedd8e-5598-4f9b-bc82-6dc1a954b362'::uuid,
  'Web 播放端首页',
  '播放端首页：Hero · 继续观看 · 分区浏览 · 智能推荐',
  'published',
  '{
    "theme": "dark",
    "global": { "layout_schema": "web-v2" },
    "rows": [
      {
        "id": "hero-web",
        "type": "hero-banner",
        "title": "精选推荐",
        "card_style": "banner",
        "visible": true,
        "source": { "type": "recommend-algorithm", "params": { "algo": "hot", "limit": 8 } }
      },
      {
        "id": "continue-web",
        "type": "shelf",
        "title": "继续观看",
        "subtitle": "从上次停下的地方继续",
        "card_style": "landscape",
        "visible": true,
        "source": { "type": "continue-watching", "params": { "limit": 12 } }
      },
      {
        "id": "banner-live",
        "type": "text-banner",
        "title": "📡 电视直播",
        "subtitle": "浏览 IPTV 频道与推流直播，支持按栏目筛选",
        "visible": true,
        "config": { "action": { "type": "live", "label": "进入直播" } },
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "divider-recent",
        "type": "divider",
        "title": "最新更新",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "recent-web",
        "type": "shelf",
        "title": "最近入库",
        "subtitle": "刚刚加入媒体库的内容",
        "card_style": "poster",
        "visible": true,
        "source": { "type": "recently-added", "params": { "limit": 20 } }
      },
      {
        "id": "divider-hot",
        "type": "divider",
        "title": "热门推荐",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "hot-web",
        "type": "topic",
        "title": "高分影视",
        "subtitle": "库内评分最高的内容",
        "card_style": "landscape",
        "visible": true,
        "source": { "type": "trending", "params": { "limit": 16 } }
      },
      {
        "id": "divider-movies",
        "type": "divider",
        "title": "电影",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "movies-grid",
        "type": "category-grid",
        "title": "电影库",
        "card_style": "poster",
        "visible": true,
        "source": { "type": "library", "params": { "type": "movie", "sort": "rating", "limit": 24 } }
      },
      {
        "id": "divider-tv",
        "type": "divider",
        "title": "剧集",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "tv-grid",
        "type": "category-grid",
        "title": "剧集库",
        "card_style": "landscape",
        "visible": true,
        "source": { "type": "library", "params": { "type": "tvshow", "sort": "rating", "limit": 24 } }
      },
      {
        "id": "divider-guess",
        "type": "divider",
        "title": "发现更多",
        "visible": true,
        "source": { "type": "manual", "params": { "ids": [] } }
      },
      {
        "id": "guess-you-like-web",
        "type": "shelf",
        "title": "猜你喜欢",
        "subtitle": "根据观影偏好智能推荐",
        "card_style": "poster",
        "visible": true,
        "source": { "type": "guess-you-like", "params": { "limit": 16, "discover_limit": 4 } }
      }
    ]
  }'::jsonb,
  1,
  NOW(),
  NOW(),
  NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM layouts
  WHERE id = 'd7cedd8e-5598-4f9b-bc82-6dc1a954b362'::uuid
    OR (config->'global'->>'layout_schema' = 'web-v2')
);

-- 确保 Web 平台发布记录
UPDATE layout_publications
SET enabled = FALSE, updated_at = NOW()
WHERE target_platform = 'web'
  AND layout_id <> 'd7cedd8e-5598-4f9b-bc82-6dc1a954b362'::uuid
  AND enabled = TRUE;

INSERT INTO layout_publications (layout_id, version, target_platform, traffic_split, enabled, created_at, updated_at)
SELECT
  l.id,
  l.version,
  'web',
  '{}'::jsonb,
  TRUE,
  NOW(),
  NOW()
FROM layouts l
WHERE l.id = 'd7cedd8e-5598-4f9b-bc82-6dc1a954b362'::uuid
  AND l.config->'global'->>'layout_schema' = 'web-v2'
  AND NOT EXISTS (
    SELECT 1 FROM layout_publications p
    WHERE p.layout_id = l.id AND p.target_platform = 'web' AND p.enabled = TRUE
  );
