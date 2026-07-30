package project

type Project struct {
	UserID         uint
	ID             uint
	Name           string
	GameType       GameType `json:"gameType"`
	ViewType       ViewType `json:"viewType"`
	TargetPlatform PlatformType
	Description    string
	Reference      string
	Style          string
}

// ProjectUpdate contains only fields supplied by a partial project update.
// A nil field is left unchanged; a non-nil field is written, including an empty value.
type ProjectUpdate struct {
	ID             uint
	Name           *string
	GameType       *GameType
	ViewType       *ViewType
	TargetPlatform *PlatformType
	Description    *string
	Reference      *string
	Style          *string
}
