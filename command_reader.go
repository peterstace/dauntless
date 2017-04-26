package main

type CommandReader interface {
	KeyPress(Key, CommandHandler)
	SetMode(CommandMode)
	Enabled() bool
	Clear()
	GetText() string
	GetCursorPos() int
	OverlaySwatch() bool
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
		}
	}
}

func (c *commandReader) SetMode(mode CommandMode) {
	c.mode = mode
}

func (c *commandReader) Enabled() bool {
	return c.mode != nil
}

func (c *commandReader) Clear() {
	c.mode = nil
	c.text = ""
	c.pos = 0
}

func (c *commandReader) GetText() string {
	return c.mode.Prompt() + c.text
}

func (c *commandReader) GetCursorPos() int {
	return len(c.mode.Prompt()) + c.pos
}

func (c *commandReader) OverlaySwatch() bool {
	return c.mode == colour{}
}
