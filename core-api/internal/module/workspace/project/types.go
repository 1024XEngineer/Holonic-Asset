package project

import perspectivedomain "github.com/1024XEngineer/Holonic-Asset/internal/module/workspace/perspective"

type Perspective = perspectivedomain.Perspective
type PlatformType string

const (
	PerspectiveTopDown   = perspectivedomain.TopDown
	PerspectiveSideOn    = perspectivedomain.SideOn
	PerspectiveIsometric = perspectivedomain.Isometric

	PlatformTypePC     PlatformType = "PC"
	PlatformTypeMobile PlatformType = "Mobile"
	PlatformTypeWeb    PlatformType = "Web"
)
