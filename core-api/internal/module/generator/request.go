package generator

import "encoding/json"

// Request captures the business intent accepted by Generator.
// Kind-specific parameters remain bounded data interpreted by Generator.
type Request struct {
	ProjectID        uint
	AssetID          *uint
	Kind             TaskType
	CreativeBrief    string
	TargetAssetPaths []string
	Parameters       json.RawMessage
}
