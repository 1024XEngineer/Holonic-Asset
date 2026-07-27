package project

type GameType string
type ViewType string
type PlatformType string

const (
	GameTypeRPG   GameType = "RPG"
	GameTypeACT   GameType = "ACT"
	GameTypeSLG   GameType = "SLG"
	GameTypeOther GameType = "Other"

	ViewTypeTopDown   ViewType = "TopDown"
	ViewTypeSideView  ViewType = "SideView"
	ViewTypeIsometric ViewType = "Isometric"
	ViewTypeOther     ViewType = "Other"

	PlatformTypePC     PlatformType = "PC"
	PlatformTypeMobile PlatformType = "Mobile"
	PlatformTypeWeb    PlatformType = "Web"
)
