-- 000003_add_updated_at_columns.up.sql
-- 000001_init_schema 建表时遗漏了部分 timestamp 列，导致 GORM INSERT 时
-- (created_at, updated_at, deleted_at) 三件套报错：
--   ERROR: column "updated_at" of relation "profiles" does not exist
--   ERROR: column "created_at" of relation "recommendations" does not exist
--
-- 涉及表：
--   profiles / seasons / episodes / layout_publications /
--   favorites / scrape_logs / recommendations —— 缺 updated_at
--   recommendations —— 还缺 created_at
--
-- 用 IF NOT EXISTS 保证幂等。

ALTER TABLE profiles            ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE seasons             ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE episodes            ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE layout_publications ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE favorites           ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE scrape_logs         ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE recommendations     ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE recommendations     ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();