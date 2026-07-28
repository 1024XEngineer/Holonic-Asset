package config

import "time"

type DBConfig struct {
	DSN             string        `mapstructure:"dsn" yaml:"dsn"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns" yaml:"maxIdleConns"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns" yaml:"maxOpenConns"`
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime" yaml:"connMaxIdleTime"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime" yaml:"connMaxLifetime"`
}

type RiverConfig struct {
	DatabaseURL   string        `mapstructure:"databaseURL" yaml:"databaseURL"`
	MaxWorkers    int           `mapstructure:"maxWorkers" yaml:"maxWorkers"`
	ClientTimeout time.Duration `mapstructure:"clientTimeout" yaml:"clientTimeout"`
}

type LogConfig struct {
	Path       string `mapstructure:"path" yaml:"path"`
	MaxSize    int    `mapstructure:"maxSize" yaml:"maxSize"`
	MaxBackups int    `mapstructure:"maxBackups" yaml:"maxBackups"`
	MaxAge     int    `mapstructure:"maxAge" yaml:"maxAge"`
	Compress   bool   `mapstructure:"compress" yaml:"compress"`
}

type Config struct {
	DB    DBConfig    `mapstructure:"db" yaml:"db"`
	River RiverConfig `mapstructure:"river" yaml:"river"`
	Log   LogConfig   `mapstructure:"log" yaml:"log"`
}
