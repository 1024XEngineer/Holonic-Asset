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
