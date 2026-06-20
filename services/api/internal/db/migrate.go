// Package db - SQL migration runner
//
// 启动时按顺序执行 migrations/sql/*.up.sql
// 用 schema_migrations 表追踪已执行版本
package db

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mediahub/api/pkg/logger"

	"gorm.io/gorm"
)

//go:embed sql/*.sql
var migrationsFS embed.FS

// Migrate 执行所有未跑过的 migration
func (d *DB) Migrate(ctx context.Context) error {
	// 1. 初始化迁移追踪表
	if err := d.DB.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`).Error; err != nil {
		return fmt.Errorf("创建 schema_migrations 失败: %w", err)
	}

	// 2. 读取已执行的版本
	applied := map[string]bool{}
	rows, err := d.DB.WithContext(ctx).Raw("SELECT version FROM schema_migrations").Rows()
	if err != nil {
		return fmt.Errorf("查询已执行迁移失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}

	// 3. 读取所有 migration 文件
	entries, err := migrationsFS.ReadDir("sql")
	if err != nil {
		return fmt.Errorf("读取 migration 目录失败: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// 4. 按顺序执行未跑过的
	for _, f := range files {
		version := strings.TrimSuffix(f, ".up.sql")
		if applied[version] {
			logger.Debug("跳过已执行迁移", "version", version)
			continue
		}

		logger.Info("执行迁移", "version", version)

		content, err := migrationsFS.ReadFile("sql/" + f)
		if err != nil {
			return fmt.Errorf("读取 %s 失败: %w", f, err)
		}

		// 在事务中执行
		err = d.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// PostgreSQL prepared statement 不支持多语句 (SQLSTATE 42601)，
			// 必须把 SQL 文件按 ; 拆开逐条 Exec。
			statements := splitSQL(string(content))
			for i, stmt := range statements {
				if err := tx.Exec(stmt).Error; err != nil {
					return fmt.Errorf("执行 %s 第 %d 条语句失败: %w", f, i+1, err)
				}
			}
			return tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
				version, time.Now()).Error
		})
		if err != nil {
			return err
		}

		logger.Info("迁移完成", "version", version)
	}

	return nil
}

// splitSQL 把多语句 SQL 文件拆成单条语句。
//
// 简化实现：按 ; split + 过滤纯注释/空行 + TrimSpace。
// 当前 migration 文件不含 PL/pgSQL 函数或字符串字面量里的 ;，
// 所以这种简单切分够用。如果未来 migration 包含复杂 PL/pgSQL，
// 建议换成 vitess/sqlparser 或 golang-migrate。
func splitSQL(content string) []string {
	var stmts []string
	for _, part := range strings.Split(content, ";") {
		// 去掉单行注释
		var cleaned []string
		for _, line := range strings.Split(part, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				continue
			}
			cleaned = append(cleaned, line)
		}
		if s := strings.TrimSpace(strings.Join(cleaned, "\n")); s != "" {
			stmts = append(stmts, s)
		}
	}
	return stmts
}
