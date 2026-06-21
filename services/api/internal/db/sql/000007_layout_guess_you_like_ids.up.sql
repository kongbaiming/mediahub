-- 000007_layout_guess_you_like_ids.up.sql
-- 按布局 ID 补插「猜你喜欢」（000006 未执行时的兜底）

UPDATE layouts
SET
  config = jsonb_set(
    config,
    '{rows}',
    jsonb_insert(
      COALESCE(config->'rows', '[]'::jsonb),
      '{1}',
      '{"id":"guess-you-like-web","type":"shelf","title":"猜你喜欢","subtitle":"根据你的观影习惯推荐","card_style":"poster","source":{"type":"guess-you-like","params":{"limit":20,"discover_limit":6}},"visible":true}'::jsonb,
      true
    )
  ),
  version = version + 1,
  updated_at = NOW()
WHERE id = 'd7cedd8e-5598-4f9b-bc82-6dc1a954b362'
  AND NOT EXISTS (
    SELECT 1 FROM jsonb_array_elements(COALESCE(config->'rows', '[]'::jsonb)) e
    WHERE e->>'id' LIKE 'guess-you-like%' OR e->'source'->>'type' = 'guess-you-like'
  );

UPDATE layouts
SET
  config = jsonb_set(
    config,
    '{rows}',
    jsonb_insert(
      COALESCE(config->'rows', '[]'::jsonb),
      '{1}',
      '{"id":"guess-you-like-tv","type":"shelf","title":"猜你喜欢","subtitle":"根据观影习惯智能推荐","card_style":"landscape","source":{"type":"guess-you-like","params":{"limit":16,"discover_limit":4}},"visible":true}'::jsonb,
      true
    )
  ),
  version = version + 1,
  updated_at = NOW()
WHERE id = 'ed9e7c99-5260-4fe4-960f-eabe764ba145'
  AND NOT EXISTS (
    SELECT 1 FROM jsonb_array_elements(COALESCE(config->'rows', '[]'::jsonb)) e
    WHERE e->>'id' LIKE 'guess-you-like%' OR e->'source'->>'type' = 'guess-you-like'
  );
