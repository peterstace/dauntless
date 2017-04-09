package main

import "os"

type SpecialKey byte

const (
	UpArrowKey SpecialKey = iota
	DownArrowKey
	RightArrowKey
	LeftArrowKey
	HomeKey
	InsertKey
	DeleteKey
	EndKey
	PageUpKey
	PageDownKey
)

func (k SpecialKey) String() string {
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
		assert(false)
		return "???"
	}
}

func collectInput(r Reactor, a App) {
	go func() {
		for {
			var buf [8]byte
			n, err := os.Stdin.Read(buf[:])
			if err != nil {
				r.Stop(err)
				return
			}
			if n == 1 {
				b := buf[0]
				r.Enque(func() { a.KeyPress(b) })
			} else if key, ok := decodeSpecialKey(buf[:n]); ok {
				r.Enque(func() { a.SpecialKeyPress(key) })
			} else {
				r.Enque(func() { a.UnknownKeySequence(buf[:n]) })
			}
		}
	}()
}

func decodeSpecialKey(buf []byte) (SpecialKey, bool) {

	if len(buf) == 3 && buf[0] == '\x1b' && buf[1] == '[' {
		switch buf[2] {
		case 'A':
			return UpArrowKey, true
		case 'B':
			return DownArrowKey, true
		case 'C':
			return RightArrowKey, true
		case 'D':
			return LeftArrowKey, true
		}
	}

	if len(buf) == 4 && buf[0] == '\x1b' && buf[1] == '[' && buf[3] == '~' {
		switch buf[2] {
		case '1':
			return HomeKey, true
		case '2':
			return InsertKey, true
		case '3':
			return DeleteKey, true
		case '4':
			return EndKey, true
		case '5':
			return PageUpKey, true
		case '6':
			return PageDownKey, true
		}
	}

	return 0, false
}
