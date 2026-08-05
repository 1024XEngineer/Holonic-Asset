package project

type GameType string
type Perspective string
type PlatformType string

const (
	GameTypeRPG GameType = "RPG"
	GameTypeACT GameType = "ACT"
	GameTypeSLG GameType = "SLG"

	PerspectiveTopDown   Perspective = "TopDown"
	PerspectiveSideOn    Perspective = "SideOn"
	PerspectiveIsometric Perspective = "Isometric"

	PlatformTypePC     PlatformType = "PC"
	PlatformTypeMobile PlatformType = "Mobile"
	PlatformTypeWeb    PlatformType = "Web"
)
