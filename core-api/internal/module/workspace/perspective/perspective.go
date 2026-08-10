package perspective

type Perspective string

const (
	TopDown   Perspective = "Top-Down"
	SideOn    Perspective = "Side-On"
	Isometric Perspective = "Isometric"
)

var supported = [...]Perspective{TopDown, SideOn, Isometric}

func Values() []Perspective {
	return append([]Perspective(nil), supported[:]...)
}

func (p Perspective) Valid() bool {
	for _, supportedPerspective := range supported {
		if p == supportedPerspective {
			return true
		}
	}
	return false
}

// CharacterDirectionCount returns the fixed sheet size for a character perspective.
func (p Perspective) CharacterDirectionCount() uint {
	switch p {
	case SideOn:
		return 2
	case TopDown:
		return 4
	case Isometric:
		return 8
	default:
		return 0
	}
}
