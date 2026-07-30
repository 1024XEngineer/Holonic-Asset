package viperx

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfig(path string, target any) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("viperx: config file path is required")
	}
	if target == nil {
		return errors.New("viperx: config target is nil")
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Pointer || targetValue.IsNil() {
		return errors.New("viperx: config target must be a non-nil pointer")
	}

	reader := viper.New()
	reader.SetConfigFile(path)
	if err := reader.ReadInConfig(); err != nil {
		return fmt.Errorf("viperx: read config file %q: %w", path, err)
	}
	if err := reader.UnmarshalExact(target); err != nil {
		return fmt.Errorf("viperx: decode config file %q: %w", path, err)
	}

	return nil
}
