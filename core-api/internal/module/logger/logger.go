package logger

import (
	defLog "log"
	"os"

	"go.uber.org/zap"
)

const defaultLogPrefix = "[Holonic Asset]"

type ZapLogger struct {
	l *zap.Logger
}

func NewLogger(l *zap.Logger) Logger {
	return &ZapLogger{l: l}
}

func (z *ZapLogger) Debug(msg string, args ...Field) {
	z.l.Debug(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Info(msg string, args ...Field) {
	z.l.Info(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Warn(msg string, args ...Field) {
	z.l.Warn(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Error(msg string, args ...Field) {
	z.l.Error(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Sync() error {
	return z.l.Sync()
}

func (z *ZapLogger) toArgs(args []Field) []zap.Field {
	res := make([]zap.Field, 0, len(args))
	for _, arg := range args {
		res = append(res, zap.Any(arg.Key, arg.Val))
	}
	return res
}

// DefaultLogger is a temporary substitute used for tests and some plugin initialization.
type DefaultLogger struct {
	log *defLog.Logger
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		log: defLog.New(os.Stderr, defaultLogPrefix, defLog.LstdFlags),
	}
}

func (d *DefaultLogger) WithField(key string, value any) any {
	return d
}

func (d *DefaultLogger) Info(msg string, args ...Field) {
	d.log.Print(msg)
}

func (d *DefaultLogger) Debug(msg string, args ...Field) {
	d.log.Print(msg)
}

func (d *DefaultLogger) Warn(msg string, args ...Field) {
	d.log.Print(msg)
}

func (d *DefaultLogger) Error(msg string, args ...Field) {
	d.log.Print(msg)
}

func (d *DefaultLogger) Sync() error {
	return nil
}
