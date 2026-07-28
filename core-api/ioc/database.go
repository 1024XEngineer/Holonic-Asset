package ioc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/1024XEngineer/Holonic-Asset/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
	"github.com/1024XEngineer/Holonic-Asset/pkg/logger"
)

func InitDB(ctx context.Context, cfg *config.DBConfig, l logger.Logger) (*gorm.DB, error) {
	if cfg == nil {
		return nil, errors.New("ioc: database config is nil")
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, errors.New("ioc: database DSN is required")
	}
	if l == nil {
		l = logger.NewDefaultLogger()
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: gormlogger.New(gormLoggerFunc(l.Debug), gormlogger.Config{
			SlowThreshold: 0,
			LogLevel:      gormlogger.Info,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("ioc: open PostgreSQL database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("ioc: get PostgreSQL connection pool: %w", err)
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
		return nil, fmt.Errorf("ioc: ping PostgreSQL database: %w", err)
	}
	if err := dao.InitTables(db.WithContext(ctx)); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ioc: initialize database tables: %w", err)
	}

	return db, nil
}

type gormLoggerFunc func(msg string, fields ...logger.Field)

func (g gormLoggerFunc) Printf(format string, args ...any) {
	g(fmt.Sprintf(format, args...), logger.Any("args", args))
}
