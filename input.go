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
		var buf []byte
		for {
			var readIn [8]byte
			n, err := os.Stdin.Read(readIn[:])
			if err != nil {
				r.Stop(err)
				return
			}
			buf = append(buf, readIn[:n]...)
			for len(buf) > 0 {
				if len(buf) == 1 && buf[0] == '\x1b' {
					// Do nothing. Wait for the next input char to decide what to do.
					break
				} else if buf[0] == '\x1b' && buf[1] == '[' {
					// Process a multi char sequence.
					foundEnd := false
					for i := 1; i < len(buf); i++ {
						if (buf[i] >= 'A' && buf[i] <= 'Z') || buf[i] == '~' {
							foundEnd = true
							if key, ok := decodeSpecialKey(buf[:i+1]); ok {
								r.Enque(func() { a.SpecialKeyPress(key) })
							} else {
								unknownKey := make([]byte, i+1)
								copy(unknownKey, buf[:i+1])
								r.Enque(func() { a.UnknownKeySequence(unknownKey) })
							}
							buf = buf[i+1:]
						}
					}
					if !foundEnd {
						break
					}
				} else {
					// Process the chars normally.
					b := buf[0]
					buf = buf[1:]
					r.Enque(func() { a.KeyPress(b) })
				}
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
