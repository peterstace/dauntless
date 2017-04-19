package main

import "fmt"

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
)

func (k Key) String() string {
	assert(len(k) != 0)
	if len(k) == 1 {
		if k[0] >= ' ' && k[0] <= '~' {
			return string(k)
		} else {
			return fmt.Sprintf("0x%02X", k[0])
		}
	}
	switch k {
	case UpArrowKey:
		return "UpArrowKey"
	case DownArrowKey:
		return "DownArrowKey"
	case RightArrowKey:
		return "RightArrowKey"
	case LeftArrowKey:
		return "LeftArrowKey"
	case HomeKey:
		return "HomeKey"
	case InsertKey:
		return "InsertKey"
	case DeleteKey:
		return "DeleteKey"
	case EndKey:
		return "EndKey"
	case PageUpKey:
		return "PageUpKey"
	case PageDownKey:
		return "PageDownKey"
	default:
		return fmt.Sprintf("%v", []byte(k))
	}
}
