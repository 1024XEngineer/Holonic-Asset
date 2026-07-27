package config

import "time"

type RiverConfig struct {
	DatabaseURL   string
	MaxWorkers    int
	ClientTimeout time.Duration
}
