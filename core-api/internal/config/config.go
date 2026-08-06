package config

import "time"

type DBConfig struct {
	DSN             string        `mapstructure:"dsn" yaml:"dsn"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns" yaml:"maxIdleConns"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns" yaml:"maxOpenConns"`
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime" yaml:"connMaxIdleTime"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime" yaml:"connMaxLifetime"`
}

type QueueConfig struct {
	DatabaseURL string        `mapstructure:"databaseURL" yaml:"databaseURL"`
	MaxWorkers  int           `mapstructure:"maxWorkers" yaml:"maxWorkers"`
	JobTimeout  time.Duration `mapstructure:"jobTimeout" yaml:"jobTimeout"`
}

type LogConfig struct {
	Path       string `mapstructure:"path" yaml:"path"`
	MaxSize    int    `mapstructure:"maxSize" yaml:"maxSize"`
	MaxBackups int    `mapstructure:"maxBackups" yaml:"maxBackups"`
	MaxAge     int    `mapstructure:"maxAge" yaml:"maxAge"`
	Compress   bool   `mapstructure:"compress" yaml:"compress"`
}

type ImageClientConfig struct {
	BaseURL      string `mapstructure:"baseURL" yaml:"baseURL"`
	APIKey       string `mapstructure:"apiKey" yaml:"apiKey"`
	DefaultModel string `mapstructure:"defaultModel" yaml:"defaultModel"`
}

type VideoClientConfig struct {
	BaseURL string `mapstructure:"baseURL" yaml:"baseURL"`
	APIKey  string `mapstructure:"apiKey" yaml:"apiKey"`
}

type QiniuConfig struct {
	AccessKey         string
	SecretKey         string
	Bucket            string
	Domain            string
	UploadURL         string
	UploadTokenExpiry time.Duration
}

type Config struct {
	DB    DBConfig          `mapstructure:"db" yaml:"db"`
	Queue QueueConfig       `mapstructure:"queue" yaml:"queue"`
	Log   LogConfig         `mapstructure:"log" yaml:"log"`
	Image ImageClientConfig `mapstructure:"image" yaml:"image"`
	Video VideoClientConfig `mapstructure:"video" yaml:"video"`
	QiNiu QiniuConfig       `mapstructure:"qiniu" yaml:"qiniu"`
}
