-- 000020_layout_ranking_topic_immersive.up.sql
-- 为 Web 首页启用沉浸式模版，并插入榜单 / 专题行

UPDATE layouts
SET
  description = COALESCE(NULLIF(description, ''), '沉浸式首页：Hero · 榜单 · 专题 · 继续观看'),
  config = '{
    "theme": "dark",
    "global": {
      "layout_schema": "immersive",
      "cms_preset": "ranking-topic-immersive-v1"
    },
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
        "id": "ranking-hot",
        "type": "ranking",
        "title": "热门榜单",
        "subtitle": "评分 TOP 10",
        "card_style": "landscape",
        "visible": true,
        "source": { "type": "trending", "params": { "limit": 10, "sort": "rating" } },
        "config": { "show_rank": true }
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
        "id": "topic-featured",
        "type": "topic",
        "title": "本周专题",
        "subtitle": "在 CMS 绑定专题专辑后展示沉浸式头图",
        "card_style": "landscape",
        "visible": true,
        "source": { "type": "album", "params": { "album_id": "", "limit": 12 } },
        "config": { "display": "immersive" }
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
        "id": "ranking-movies",
        "type": "ranking",
        "title": "电影榜",
        "subtitle": "高分电影",
        "card_style": "poster",
        "visible": true,
        "source": { "type": "library", "params": { "type": "movie", "sort": "rating", "limit": 10 } },
        "config": { "show_rank": true }
      },
      {
        "id": "ranking-tv",
        "type": "ranking",
        "title": "剧集榜",
        "subtitle": "口碑剧集",
        "card_style": "poster",
        "visible": true,
        "source": { "type": "library", "params": { "type": "tvshow", "sort": "rating", "limit": 10 } },
        "config": { "show_rank": true }
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
WHERE id = 'd7cedd8e-5598-4f9b-bc82-6dc1a954b362'::uuid
  AND (config->'global'->>'cms_preset' IS DISTINCT FROM 'ranking-topic-immersive-v1');

-- 兼容：其他仍为 web-v2 的已发布 Web 布局也升级 schema（仅改 global，保留自定义行）
UPDATE layouts
SET
  config = jsonb_set(
    jsonb_set(
      config,
      '{global,layout_schema}',
      '"immersive"'::jsonb
    ),
    '{global,cms_preset}',
    '"ranking-topic-immersive-v1"'::jsonb
  ),
  version = version + 1,
  updated_at = NOW()
WHERE config->'global'->>'layout_schema' = 'web-v2'
  AND (config->'global'->>'cms_preset' IS DISTINCT FROM 'ranking-topic-immersive-v1')
  AND id <> 'd7cedd8e-5598-4f9b-bc82-6dc1a954b362'::uuid;
