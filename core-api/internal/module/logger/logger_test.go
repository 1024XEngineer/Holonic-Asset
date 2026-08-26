package logger

import (
	"errors"
	"reflect"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestFieldConstructors(t *testing.T) {
	err := errors.New("sample error")

	tests := []struct {
		name string
		got  Field
		want Field
	}{
		{
			name: "Any",
			got:  Any("meta", map[string]int{"a": 1}),
			want: Field{Key: "meta", Val: map[string]int{"a": 1}},
		},
		{
			name: "Error",
			got:  Error(err),
			want: Field{Key: "errorx", Val: err},
		},
		{
			name: "Int64",
			got:  Int64("count", 1234567890123),
			want: Field{Key: "count", Val: int64(1234567890123)},
		},
		{
			name: "Int",
			got:  Int("num", 42),
			want: Field{Key: "num", Val: 42},
		},
		{
			name: "String",
			got:  String("name", "holonic"),
			want: Field{Key: "name", Val: "holonic"},
		},
		{
			name: "Int32",
			got:  Int32("id", 101),
			want: Field{Key: "id", Val: int32(101)},
		},
		{
			name: "Float32",
			got:  Float32("ratio", 3.14),
			want: Field{Key: "ratio", Val: float32(3.14)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got.Key != tt.want.Key {
				t.Errorf("got Key %q, want %q", tt.got.Key, tt.want.Key)
			}
			if !reflect.DeepEqual(tt.got.Val, tt.want.Val) {
				t.Errorf("got Val %v, want %v", tt.got.Val, tt.want.Val)
			}
		})
	}
}

func TestZapLogger(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	zapL := zap.New(core)
	l := NewLogger(zapL)

	l.Debug("debug message", String("k1", "v1"))
	l.Info("info message", Int("k2", 2))
	l.Warn("warn message", Int32("k3", 3))
	l.Error("error message", Error(errors.New("fail")))

	if err := l.Sync(); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}

	entries := logs.All()
	if len(entries) != 4 {
		t.Fatalf("expected 4 logs, got %d", len(entries))
	}

	if entries[0].Level != zapcore.DebugLevel || entries[0].Message != "debug message" {
		t.Errorf("unexpected debug entry: %+v", entries[0])
	}
	if len(entries[0].Context) != 1 || entries[0].Context[0].Key != "k1" || entries[0].Context[0].String != "v1" {
		t.Errorf("unexpected debug fields: %+v", entries[0].Context)
	}

	if entries[1].Level != zapcore.InfoLevel || entries[1].Message != "info message" {
		t.Errorf("unexpected info entry: %+v", entries[1])
	}

	if entries[2].Level != zapcore.WarnLevel || entries[2].Message != "warn message" {
		t.Errorf("unexpected warn entry: %+v", entries[2])
	}

	if entries[3].Level != zapcore.ErrorLevel || entries[3].Message != "error message" {
		t.Errorf("unexpected error entry: %+v", entries[3])
	}
}

func TestDefaultLogger(t *testing.T) {
	dl := NewDefaultLogger()

	if dl == nil {
		t.Fatal("expected non-nil default logger")
	}

	chained := dl.WithField("trace", "123")
	if chained != dl {
		t.Errorf("expected WithField to return receiver, got %v", chained)
	}

	dl.Debug("default debug", String("k", "v"))
	dl.Info("default info", Int("num", 1))
	dl.Warn("default warn", Error(errors.New("w")))
	dl.Error("default error", Any("obj", 123))

	if err := dl.Sync(); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}
}
