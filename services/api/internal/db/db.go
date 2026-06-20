// Package db 负责数据库连接与生命周期管理
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/mediahub/api/internal/domain/history"
	"github.com/mediahub/api/internal/domain/layout"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/domain/user"

	gormlogger "gorm.io/gorm/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB 是数据库句柄的封装
type DB struct {
	*gorm.DB
}

// PoolStats 连接池统计
type PoolStats struct {
	OpenConnections int
	InUse           int
	Idle            int
	WaitCount       int64
	WaitDuration    time.Duration
}

// Connect 建立数据库连接（带重试 + Pool 配置 + 慢查询日志）
func Connect(dsn string, debug bool) (*DB, error) {
	cfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		NowFunc: func() time.Time { return time.Now().UTC() },
		PrepareStmt: true, // 预编译 SQL，提升性能
	}

	if debug {
		cfg.Logger = gormlogger.Default.LogMode(gormlogger.Info)
	}

	gdb, err := gorm.Open(postgres.Open(dsn), cfg)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}

	// 连接池配置（DS920+ 4-8GB RAM 友好）
	// - MaxOpen=25: 给 Asynq workers / GORM / 业务流量三方面预留
	// - MaxIdle=10: 避免频繁创建/销毁
	// - ConnMaxLifetime=1h: 配合云 RDS / NAS 的连接回收
	// - ConnMaxIdleTime=10m: 空闲连接释放
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("数据库 ping 失败: %w", err)
	}

	return &DB{DB: gdb}, nil
}

// Stats 返回连接池统计信息（用于健康检查）
func (d *DB) Stats() PoolStats {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return PoolStats{}
	}
	s := sqlDB.Stats()
	return PoolStats{
		OpenConnections: s.OpenConnections,
		InUse:           s.InUse,
		Idle:            s.Idle,
		WaitCount:       s.WaitCount,
		WaitDuration:    s.WaitDuration,
	}
}

// AutoMigrate 自动迁移（开发用，生产建议用 SQL migration）
func (d *DB) AutoMigrate() error {
	return d.DB.AutoMigrate(
		&media.Media{},
		&media.Season{},
		&media.Episode{},
		&layout.Layout{},
		&layout.Publication{},
		&user.User{},
		&user.Profile{},
		&history.History{},
		&history.Favorite{},
	)
}

// Ping 健康检查
func (d *DB) Ping(ctx context.Context) error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close 关闭连接
func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
