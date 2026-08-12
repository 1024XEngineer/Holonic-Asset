package asset

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrVersionConflict = errors.New("asset version conflict")

type AssetRecord struct {
	ID              uint
	AssetID         uint
	Version         uint
	ExpectedVersion uint
	ContentID       uint
	CreatedAt       time.Time
	Name            string
	Description     string
	Perspective     Perspective
	Dimensions      json.RawMessage
	Content         json.RawMessage
}
