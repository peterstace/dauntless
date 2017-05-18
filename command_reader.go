package main

type CommandReader interface {
	KeyPress(Key, CommandHandler)
	SetMode(CommandMode)
	Clear()

	EnteredText() string
	Pos() int
}

type commandReader struct {
	mode CommandMode
	text string
	pos  int
}

func (c *commandReader) KeyPress(k Key, h CommandHandler) {

	assert(c.mode != nil)

	if len(k) == 1 {
		b := k[0]
		if b >= ' ' && b <= '~' {
			c.text = c.text[:c.pos] + string([]byte{b}) + c.text[c.pos:]
			c.pos++
		} else if b == 127 && len(c.text) >= 1 {
			c.text = c.text[:c.pos-1] + c.text[c.pos:]
			c.pos--
		} else if b == '\n' {
			c.mode.Entered(c.text, h)
			c.Clear()
		}
	} else {
		if k == LeftArrowKey {
			c.pos = max(0, c.pos-1)
		} else if k == RightArrowKey {
			c.pos = min(c.pos+1, len(c.text))
		} else if k == DeleteKey && c.pos < len(c.text) {
			c.text = c.text[:c.pos] + c.text[c.pos+1:]
		} else if k == HomeKey {
			c.pos = 0
		} else if k == EndKey {
			c.pos = len(c.text)
		}
	}
}

func (c *commandReader) SetMode(mode CommandMode) {
	c.mode = mode
}

func (c *commandReader) Clear() {
	c.mode = nil
	c.text = ""
	c.pos = 0
}

func (c *commandReader) EnteredText() string {
	return c.text
}

func (c *commandReader) Pos() int {
	return c.pos
}
