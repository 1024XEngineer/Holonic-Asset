package project

import perspectivedomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/perspective"

type GameType string
type Perspective = perspectivedomain.Perspective
type PlatformType string

const (
	GameTypeRPG GameType = "RPG"
	GameTypeACT GameType = "ACT"
	GameTypeSLG GameType = "SLG"

	PerspectiveTopDown   = perspectivedomain.TopDown
	PerspectiveSideOn    = perspectivedomain.SideOn
	PerspectiveIsometric = perspectivedomain.Isometric

	PlatformTypePC     PlatformType = "PC"
	PlatformTypeMobile PlatformType = "Mobile"
	PlatformTypeWeb    PlatformType = "Web"
)
