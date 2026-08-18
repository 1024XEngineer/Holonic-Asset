package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm/schema"
)

const assetTagsSerializerName = "asset_tags"

func init() {
	schema.RegisterSerializer(assetTagsSerializerName, assetTagsSerializer{})
}

// assetTagsSerializer stores tags as JSON text and tolerates the legacy scalar
// values written by the old map-based update path.
type assetTagsSerializer struct{}

func (assetTagsSerializer) Scan(
	ctx context.Context,
	field *schema.Field,
	dst reflect.Value,
	dbValue any,
) error {
	fieldValue := reflect.New(field.FieldType)
	tags, err := decodeAssetTags(dbValue)
	if err != nil {
		return err
	}
	fieldValue.Elem().Set(reflect.ValueOf(tags))
	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

func (assetTagsSerializer) Value(
	_ context.Context,
	_ *schema.Field,
	_ reflect.Value,
	fieldValue any,
) (any, error) {
	encoded, err := json.Marshal(fieldValue)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

func decodeAssetTags(dbValue any) ([]string, error) {
	if dbValue == nil {
		return nil, nil
	}

	var raw []byte
	switch value := dbValue.(type) {
	case []byte:
		raw = append([]byte(nil), value...)
	case string:
		raw = []byte(value)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("dao: decode asset tags: %w", err)
		}
		raw = encoded
	}

	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return nil, nil
	}

	var tags []string
	if err := json.Unmarshal(raw, &tags); err == nil {
		return tags, nil
	}

	// Some rows were written by the previous update path as a JSON scalar
	// (or even as plain text). Preserve that value as one tag so reads remain
	// backward-compatible; the next update rewrites it as a JSON array.
	var tag string
	if err := json.Unmarshal(raw, &tag); err == nil {
		return []string{tag}, nil
	}
	return []string{value}, nil
}
