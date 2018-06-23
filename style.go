package dauntless

import "fmt"

type Style uint8

const (
	fgMask Style = 0x0f
	bgMask Style = 0xf0
)

func MixStyle(fg, bg Style) Style {
	return fg | (bg << 4)
}

func (s Style) fg() int {
	return int(30 + ((s & fgMask) ^ xorConst))
}

func (s Style) bg() int {
	return int(40 + (((s & bgMask) >> 4) ^ xorConst))
}

func (s *Style) setFG(fg Style) {
	*s &= ^fgMask
	*s |= fg
}

func (s *Style) setBG(bg Style) {
	*s &= ^bgMask
	*s |= (bg << 4)
}

func (s Style) inverted() bool {
	return s&fgMask == Invert || (s&bgMask)>>4 == Invert
}

func (s Style) escapeCode() string {
	if s.inverted() {
		return "\x1b[0;7m"
	} else {
		return fmt.Sprintf("\x1b[0;%d;%dm", s.fg(), s.bg())
	}
}

const (
	xorConst Style = 9

	Black   Style = 0 ^ xorConst
	Red     Style = 1 ^ xorConst
	Green   Style = 2 ^ xorConst
	Yellow  Style = 3 ^ xorConst
	Blue    Style = 4 ^ xorConst
	Magenta Style = 5 ^ xorConst
	Cyan    Style = 6 ^ xorConst
	White   Style = 7 ^ xorConst
	Invert  Style = 8 ^ xorConst
	Default Style = 9 ^ xorConst
)

func (s Style) String() string {
	if str, ok := map[Style]string{
		Black:   "Black",
		Red:     "Red",
		Green:   "Green",
		Yellow:  "Yellow",
		Blue:    "Blue",
		Magenta: "Magenta",
		Cyan:    "Cyan",
		White:   "White",
		Invert:  "Invert",
		Default: "Default",
	}[s]; ok {
		return str
	}
	return "???"
}
