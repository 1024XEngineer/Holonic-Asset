package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/config"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/logger"
)

func InitLogger(logConfig *config.LogConfig) (logger.Logger, error) {
	if logConfig == nil {
		return nil, errors.New("app: log config is nil")
	}
	if strings.TrimSpace(logConfig.Path) == "" {
		return nil, errors.New("app: log path is required")
	}

	// Use zap's own config struct directly for configuration.
	// Configure Lumberjack to support log file rotation.
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logConfig.Path,       // Specifies the log file path.
		MaxSize:    logConfig.MaxSize,    // Maximum size of each log file, in MB.
		MaxBackups: logConfig.MaxBackups, // Maximum number of old log files to retain.
		MaxAge:     logConfig.MaxAge,     // Maximum number of days to retain old log files.
		Compress:   logConfig.Compress,   // Whether to compress old log files.
	}
	logLevelStr := os.Getenv("LOG_LEVEL")
	logLevel := getZapLevel(logLevelStr)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := &PrettyJSONEncoder{Encoder: zapcore.NewJSONEncoder(encoderCfg)}

	// Create zap log core.
	core := zapcore.NewCore(
		encoder, //zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), // Use custom encoder for pretty-printed logs in dev environment.
		zapcore.AddSync(lumberjackLogger),
		logLevel, // Set the log level.
	)

	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	res := logger.NewLogger(l)

	if err := res.Sync(); err != nil {
		return nil, fmt.Errorf("app: initialize logger: %w", err)
	}
	return res, nil
}

func getZapLevel(levelStr string) zapcore.Level {
	levelStr = strings.ToLower(levelStr)

	switch levelStr {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		// Default to Debug level.
		return zapcore.DebugLevel
	}
}

type PrettyJSONEncoder struct {
	zapcore.Encoder
}

func (p *PrettyJSONEncoder) Clone() zapcore.Encoder {
	return &PrettyJSONEncoder{Encoder: p.Encoder.Clone()}
}

func (p *PrettyJSONEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := p.Encoder.EncodeEntry(ent, fields)
	if err != nil {
		return buf, err
	}

	var tmp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &tmp); err != nil {
		return buf, err
	}

	prettyBytes, err := json.MarshalIndent(tmp, "", "  ")
	if err != nil {
		return buf, err
	}

	buf.Reset()
	if _, err := buf.Write(prettyBytes); err != nil {
		return buf, err
	}
	if err := buf.WriteByte('\n'); err != nil {
		return buf, err
	}
	return buf, nil
}
