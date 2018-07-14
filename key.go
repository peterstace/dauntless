package main

import (
	"fmt"

	"github.com/peterstace/dauntless/assert"
)

type Key string

const (
	UpArrowKey    Key = "\x1b[A"
	DownArrowKey  Key = "\x1b[B"
	RightArrowKey Key = "\x1b[C"
	LeftArrowKey  Key = "\x1b[D"
	HomeKey       Key = "\x1b[1~"
	InsertKey     Key = "\x1b[2~"
	DeleteKey     Key = "\x1b[3~"
	EndKey        Key = "\x1b[4~"
	PageUpKey     Key = "\x1b[5~"
	PageDownKey   Key = "\x1b[6~"
	ShiftTab      Key = "\x1b[Z"
)

func (k Key) String() string {
	assert.True(len(k) != 0)
	if len(k) == 1 {
		if k[0] >= ' ' && k[0] <= '~' {
			return string(k)
		} else if k[0] == '\t' {
			return "<tab>"
		} else {
			return fmt.Sprintf("0x%02X", k[0])
		}
	}
	switch k {
	case UpArrowKey:
		return "<up-arrow>"
	case DownArrowKey:
		return "<down-arrow>"
	case RightArrowKey:
		return "<right-arrow>"
	case LeftArrowKey:
		return "<left-arrow>"
	case HomeKey:
		return "<home>"
	case InsertKey:
		return "<insert>"
	case DeleteKey:
		return "<delete>"
	case EndKey:
		return "<end>"
	case PageUpKey:
		return "<page-up>"
	case PageDownKey:
		return "<page-down>"
	case ShiftTab:
		return "<shift-tab>"
	default:
		return fmt.Sprintf("%v", []byte(k))
	}
}
