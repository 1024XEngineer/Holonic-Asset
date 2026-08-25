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

type AuthConfig struct {
	JWTSecret   string        `mapstructure:"jwtSecret" yaml:"jwtSecret"`
	TokenExpiry time.Duration `mapstructure:"tokenExpiry" yaml:"tokenExpiry"`
}

type HTTPConfig struct {
	AllowedOrigins []string `mapstructure:"allowedOrigins" yaml:"allowedOrigins"`
}

type PprofConfig struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

type LogConfig struct {
	Path       string `mapstructure:"path" yaml:"path"`
	MaxSize    int    `mapstructure:"maxSize" yaml:"maxSize"`
	MaxBackups int    `mapstructure:"maxBackups" yaml:"maxBackups"`
	MaxAge     int    `mapstructure:"maxAge" yaml:"maxAge"`
	Compress   bool   `mapstructure:"compress" yaml:"compress"`
}

// ModelConfig maps one gateway model to its required wire protocol and endpoint settings.
type ModelConfig struct {
	Name     string `mapstructure:"name" yaml:"name"`
	Protocol string `mapstructure:"protocol" yaml:"protocol"`
	BaseURL  string `mapstructure:"baseURL" yaml:"baseURL"`
	APIKey   string `mapstructure:"apiKey" yaml:"apiKey"`
}

type ImageClientConfig struct {
	BaseURL       string `mapstructure:"baseURL" yaml:"baseURL"`
	APIKey        string `mapstructure:"apiKey" yaml:"apiKey"`
	DefaultModel  string `mapstructure:"defaultModel" yaml:"defaultModel"`
	FallbackModel string `mapstructure:"fallbackModel" yaml:"fallbackModel"`
	// Provider is retained for configurations that predate model routing.
	Provider string        `mapstructure:"provider" yaml:"provider"`
	Models   []ModelConfig `mapstructure:"models" yaml:"models"`
}

type LLMClientConfig struct {
	BaseURL      string        `mapstructure:"baseURL" yaml:"baseURL"`
	APIKey       string        `mapstructure:"apiKey" yaml:"apiKey"`
	DefaultModel string        `mapstructure:"defaultModel" yaml:"defaultModel"`
	Models       []ModelConfig `mapstructure:"models" yaml:"models"`
}

type VideoClientConfig struct {
	BaseURL      string        `mapstructure:"baseURL" yaml:"baseURL"`
	APIKey       string        `mapstructure:"apiKey" yaml:"apiKey"`
	Models       []ModelConfig `mapstructure:"models" yaml:"models"`
	PollInterval time.Duration `mapstructure:"pollInterval" yaml:"pollInterval"`
	PollTimeout  time.Duration `mapstructure:"pollTimeout" yaml:"pollTimeout"`
	MaxRetries   int           `mapstructure:"maxRetries" yaml:"maxRetries"`
	RetryDelay   time.Duration `mapstructure:"retryDelay" yaml:"retryDelay"`
}

type QiniuConfig struct {
	AccessKey         string        `mapstructure:"accessKey" yaml:"accessKey"`
	SecretKey         string        `mapstructure:"secretKey" yaml:"secretKey"`
	Bucket            string        `mapstructure:"bucket" yaml:"bucket"`
	Domain            string        `mapstructure:"domain" yaml:"domain"`
	UploadURL         string        `mapstructure:"uploadURL" yaml:"uploadURL"`
	UploadTokenExpiry time.Duration `mapstructure:"uploadTokenExpiry" yaml:"uploadTokenExpiry"`
	DownloadURLExpiry time.Duration `mapstructure:"downloadURLExpiry" yaml:"downloadURLExpiry"`
}

type Config struct {
	DB    DBConfig          `mapstructure:"db" yaml:"db"`
	Queue QueueConfig       `mapstructure:"queue" yaml:"queue"`
	Auth  AuthConfig        `mapstructure:"auth" yaml:"auth"`
	HTTP  HTTPConfig        `mapstructure:"http" yaml:"http"`
	Pprof PprofConfig       `mapstructure:"pprof" yaml:"pprof"`
	Log   LogConfig         `mapstructure:"log" yaml:"log"`
	Image ImageClientConfig `mapstructure:"image" yaml:"image"`
	LLM   LLMClientConfig   `mapstructure:"llm" yaml:"llm"`
	Video VideoClientConfig `mapstructure:"video" yaml:"video"`
	QiNiu QiniuConfig       `mapstructure:"qiniu" yaml:"qiniu"`
}
