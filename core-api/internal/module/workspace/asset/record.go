package asset

import (
	"encoding/json"
	"time"
)

type AssetRecord struct {
	ID        uint
	AssetID   uint
	Version   uint
	ContentID uint
	CreatedAt time.Time
	Content   json.RawMessage
}
