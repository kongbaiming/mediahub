-- 000006_layout_guess_you_like.up.sql
-- 为家庭媒资首页 (Web/TV) 布局插入「猜你喜欢」行（位于「继续观看」之后）

UPDATE layouts
SET
  config = jsonb_set(
    config,
    '{rows}',
    CASE
      WHEN jsonb_array_length(COALESCE(config->'rows', '[]'::jsonb)) >= 1 THEN
        jsonb_insert(
          COALESCE(config->'rows', '[]'::jsonb),
          '{1}',
          '{
            "id": "guess-you-like-web",
            "type": "shelf",
            "title": "猜你喜欢",
            "subtitle": "根据你的观影习惯推荐",
            "card_style": "poster",
            "source": {
              "type": "guess-you-like",
              "params": {"limit": 20, "discover_limit": 6}
            },
            "visible": true
          }'::jsonb,
          true
        )
      ELSE
        COALESCE(config->'rows', '[]'::jsonb) || '[
          {
            "id": "guess-you-like-web",
            "type": "shelf",
            "title": "猜你喜欢",
            "subtitle": "根据你的观影习惯推荐",
            "card_style": "poster",
            "source": {
              "type": "guess-you-like",
              "params": {"limit": 20, "discover_limit": 6}
            },
            "visible": true
          }
        ]'::jsonb
    END
  ),
  version = version + 1,
  updated_at = NOW()
WHERE NOT EXISTS (
  SELECT 1
  FROM jsonb_array_elements(COALESCE(config->'rows', '[]'::jsonb)) e
  WHERE e->>'id' LIKE 'guess-you-like%'
    OR e->'source'->>'type' = 'guess-you-like'
)
AND (
  name LIKE '%家庭媒资%Web%'
  OR EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(config->'rows', '[]'::jsonb)) e
    WHERE e->>'id' = 'hero-web'
  )
);

UPDATE layouts
SET
  config = jsonb_set(
    config,
    '{rows}',
    CASE
      WHEN jsonb_array_length(COALESCE(config->'rows', '[]'::jsonb)) >= 1 THEN
        jsonb_insert(
          COALESCE(config->'rows', '[]'::jsonb),
          '{1}',
          '{
            "id": "guess-you-like-tv",
            "type": "shelf",
            "title": "猜你喜欢",
            "subtitle": "根据观影习惯智能推荐",
            "card_style": "landscape",
            "source": {
              "type": "guess-you-like",
              "params": {"limit": 16, "discover_limit": 4}
            },
            "visible": true
          }'::jsonb,
          true
        )
      ELSE
        COALESCE(config->'rows', '[]'::jsonb) || '[
          {
            "id": "guess-you-like-tv",
            "type": "shelf",
            "title": "猜你喜欢",
            "subtitle": "根据观影习惯智能推荐",
            "card_style": "landscape",
            "source": {
              "type": "guess-you-like",
              "params": {"limit": 16, "discover_limit": 4}
            },
            "visible": true
          }
        ]'::jsonb
    END
  ),
  version = version + 1,
  updated_at = NOW()
WHERE NOT EXISTS (
  SELECT 1
  FROM jsonb_array_elements(COALESCE(config->'rows', '[]'::jsonb)) e
  WHERE e->>'id' LIKE 'guess-you-like%'
    OR e->'source'->>'type' = 'guess-you-like'
)
AND (
  name LIKE '%家庭媒资%TV%'
  OR EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(config->'rows', '[]'::jsonb)) e
    WHERE e->>'id' = 'hero-tv'
  )
);
