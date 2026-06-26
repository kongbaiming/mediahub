-- 清理重复的播放历史，并添加唯一约束防止再次出现

DELETE FROM history h1
USING history h2
WHERE h1.profile_id = h2.profile_id
  AND h1.media_id = h2.media_id
  AND h1.id <> h2.id
  AND (
    (h1.episode_id IS NULL AND h2.episode_id IS NULL)
    OR h1.episode_id = h2.episode_id
  )
  AND h1.updated_at < h2.updated_at;

CREATE UNIQUE INDEX IF NOT EXISTS uq_history_profile_media_no_episode
    ON history (profile_id, media_id)
    WHERE episode_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_history_profile_media_episode
    ON history (profile_id, media_id, episode_id)
    WHERE episode_id IS NOT NULL;
