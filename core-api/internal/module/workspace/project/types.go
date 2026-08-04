package project

type GameType string
type ViewType string
type PlatformType string

const (
	GameTypeRPG GameType = "RPG"
	GameTypeACT GameType = "ACT"
	GameTypeSLG GameType = "SLG"

	ViewTypeTopDown   ViewType = "TopDown"
	ViewTypeSideView  ViewType = "SideView"
	ViewTypeIsometric ViewType = "Isometric"

	PlatformTypePC     PlatformType = "PC"
	PlatformTypeMobile PlatformType = "Mobile"
	PlatformTypeWeb    PlatformType = "Web"
)
