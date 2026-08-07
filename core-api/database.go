package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

func InitDB(ctx context.Context, cfg *config.DBConfig, l logger.Logger) (*gorm.DB, error) {
	if cfg == nil {
		return nil, errors.New("app: database config is nil")
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, errors.New("app: database DSN is required")
	}
	if l == nil {
		l = logger.NewDefaultLogger()
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: gormlogger.New(gormLoggerFunc(l.Warn), gormlogger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      gormlogger.Warn,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("app: open PostgreSQL database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("app: get PostgreSQL connection pool: %w", err)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("app: ping PostgreSQL database: %w", err)
	}
	if err := dao.InitTables(db.WithContext(ctx)); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("app: initialize database tables: %w", err)
	}
	if err := InitRiver(ctx, cfg.DSN); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return db, nil
}

func InitRiver(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("app: create River database pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("app: ping River database: %w", err)
	}

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("app: create River migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("app: migrate River tables: %w", err)
	}
	return nil
}

type gormLoggerFunc func(msg string, fields ...logger.Field)

func (g gormLoggerFunc) Printf(format string, args ...any) {
	g(fmt.Sprintf(format, args...), logger.Any("args", args))
}
