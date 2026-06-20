-- Web 播放器约定的默认 Profile（与前端 localStorage 中 ID 一致）
INSERT INTO profiles (id, user_id, name, is_kid, created_at, updated_at)
SELECT
    '00000000-0000-0000-0000-000000000001'::uuid,
    u.id,
    '默认',
    false,
    NOW(),
    NOW()
FROM users u
WHERE u.username = 'admin'
ON CONFLICT (id) DO NOTHING;
