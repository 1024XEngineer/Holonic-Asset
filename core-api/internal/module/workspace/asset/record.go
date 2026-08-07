package asset

import (
	"encoding/json"
	"time"
)

type AssetRecord struct {
	ID          uint
	AssetID     uint
	Version     uint
	ContentID   uint
	CreatedAt   time.Time
	Name        string
	Description string
	Perspective Perspective
	Scale       json.RawMessage
	Content     json.RawMessage
}
