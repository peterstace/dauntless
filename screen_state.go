package main

type ScreenState struct {
	Chars  []byte
	Styles []Style
	Cols   int
	ColPos int // Always on last row.
}

func NewScreenState(rows, cols int) ScreenState {
	s := ScreenState{Cols: cols, ColPos: cols - 1}
	n := rows * cols
	s.Chars = make([]byte, n)
	s.Styles = make([]Style, n)
	return s
}

func (s ScreenState) Init() {
	for i := range s.Chars {
		s.Chars[i] = ' '
		s.Styles[i] = 0
	}
}

func (s ScreenState) Rows() int {
	return len(s.Chars) / s.Cols
}

func (s ScreenState) RowColIdx(row, col int) int {
	return row*s.Cols + col
}

func (s ScreenState) CloneInto(into *ScreenState) {
	if len(s.Chars) != len(into.Chars) {
		into.Chars = make([]byte, len(s.Chars))
		into.Styles = make([]Style, len(s.Styles))
	}
	assert(len(s.Styles) == len(into.Styles))
	into.Cols = s.Cols
	into.ColPos = s.ColPos
	copy(into.Chars, s.Chars)
	copy(into.Styles, s.Styles)
}

func (s ScreenState) Equal(rhs ScreenState) bool {
	if s.Cols != rhs.Cols || s.ColPos != rhs.ColPos {
		return false
	}
	for i := range s.Chars {
		if s.Chars[i] != rhs.Chars[i] {
			return false
		}
	}
	for i := range s.Styles {
		if s.Styles[i] != rhs.Styles[i] {
			return false
		}
	}
	return true
}
