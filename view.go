package main

import (
	"fmt"
	"time"
)

func CreateView(m *Model) ScreenState {

	state := NewScreenState(m.rows, m.cols)
	state.Init()

	assert(len(m.fwd) == 0 || m.fwd[0].offset == m.offset)
	var lineBuf []byte
	var styleBuf []Style
	var fwdIdx int
	lineRows := m.rows - 2 // 2 rows reserved for status line and command line.
	for row := 0; row < lineRows; row++ {
		if fwdIdx < len(m.fwd) {
			usePrefix := len(lineBuf) != 0
			if len(lineBuf) == 0 {
				assert(len(styleBuf) == 0)
				data := m.fwd[fwdIdx].data
				if data[len(data)-1] == '\n' {
					data = data[:len(data)-1]
				}
				lineBuf = renderLine(data)
				regexes := m.regexes
				if m.tmpRegex != nil {
					regexes = append(regexes, regex{MixStyle(Invert, Invert), m.tmpRegex})
				}
				styleBuf = renderStyle(data, regexes)
				fwdIdx++
			}
			if !m.lineWrapMode {
				if m.xPosition < len(lineBuf) {
					copy(state.Chars[row*m.cols:(row+1)*m.cols], lineBuf[m.xPosition:])
					copy(state.Styles[row*m.cols:(row+1)*m.cols], styleBuf[m.xPosition:])
				}
				lineBuf = nil
				styleBuf = nil
			} else {
				var prefix string
				if usePrefix && len(m.config.WrapPrefix)+1 < m.cols {
					prefix = m.config.WrapPrefix
				}
				copy(state.Chars[row*m.cols:(row+1)*m.cols], prefix)
				copiedA := copy(state.Chars[row*m.cols+len(prefix):(row+1)*m.cols], lineBuf)
				copiedB := copy(state.Styles[row*m.cols+len(prefix):(row+1)*m.cols], styleBuf)
				assert(copiedA == copiedB)
				lineBuf = lineBuf[copiedA:]
				styleBuf = styleBuf[copiedB:]
			}
		} else {
			state.Chars[state.RowColIdx(row, 0)] = '~'
		}
	}

	drawStatusLine(m, state)

	state.ColPos = m.cols - 1
	commandLineText := ""
	if m.commandReader.Enabled() {
		commandLineText = m.commandReader.GetText()
		state.ColPos = min(state.ColPos, m.commandReader.GetCursorPos())
	} else {
		if time.Now().Sub(m.msgSetAt) < msgLingerDuration {
			commandLineText = m.msg
		}
	}

	commandRow := m.rows - 1
	copy(state.Chars[commandRow*m.cols:(commandRow+1)*m.cols], commandLineText)

	if m.commandReader.OverlaySwatch() {
		overlaySwatch(state)
	}

	return state
}

func renderLine(data string) []byte {
	buf := make([]byte, len(data))
	for i := range data {
		buf[i] = displayByte(data[i])
	}
	return buf
}

func renderStyle(data string, regexes []regex) []Style {

	buf := make([]Style, len(data))
	for _, regex := range regexes {
		for _, match := range regex.re.FindAllStringIndex(data, -1) {
			for i := match[0]; i < match[1]; i++ {
				buf[i] = regex.style
			}
		}
	}
	return buf
}

func drawStatusLine(m *Model, state ScreenState) {

	statusRow := m.rows - 2
	for col := 0; col < state.Cols; col++ {
		state.Styles[statusRow*m.cols+col] = MixStyle(Invert, Invert)
	}

	// Offset percentage.
	pct := float64(m.offset) / float64(m.fileSize) * 100
	var pctStr string
	switch {
	case pct < 10:
		// 9.99%
		pctStr = fmt.Sprintf("%3.2f%%", pct)
	default:
		// 99.9%
		pctStr = fmt.Sprintf("%3.1f%%", pct)
	}

	// Line wrap mode.
	var lineWrapMode string
	if m.lineWrapMode {
		lineWrapMode = "line-wrap-mode:on "
	} else {
		lineWrapMode = "line-wrap-mode:off"
	}

	currentRegexpStr := "re:<none>"
	if m.tmpRegex != nil {
		currentRegexpStr = "re(tmp):" + m.tmpRegex.String()
	} else if len(m.regexes) > 0 {
		currentRegexpStr = fmt.Sprintf("re(%d):%s", len(m.regexes), m.regexes[0].re.String())
	}

	statusRight := fmt.Sprintf("fwd:%d bck:%d ", len(m.fwd), len(m.bck)) + lineWrapMode + " " + pctStr + " "
	statusLeft := " " + m.filename + " " + currentRegexpStr

	buf := state.Chars[statusRow*m.cols : (statusRow+1)*m.cols]
	copy(buf[max(0, len(buf)-len(statusRight)):], statusRight)
	copy(buf[:], statusLeft)
}

func overlaySwatch(state ScreenState) {

	const sideBorder = 2
	const topBorder = 1
	const colourWidth = 4
	const swatchWidth = len(styles)*colourWidth + sideBorder*2
	const swatchHeight = len(styles) + topBorder*2

	startCol := (state.Cols - swatchWidth) / 2
	startRow := (state.Rows() - swatchHeight) / 2
	endCol := startCol + swatchWidth
	endRow := startRow + swatchHeight

	for row := startRow; row < endRow; row++ {
		for col := startCol; col < endCol; col++ {
			idx := state.RowColIdx(row, col)
			if col-startCol < 2 || endCol-col <= 2 || row-startRow < 1 || endRow-row <= 1 {
				state.Styles[idx] = MixStyle(Invert, Invert)
			}
			state.Chars[idx] = ' '
		}
	}

	for fg := 0; fg < len(styles); fg++ {
		for bg := 0; bg < len(styles); bg++ {
			start := startCol + sideBorder + bg*colourWidth
			row := startRow + topBorder + fg
			state.Chars[state.RowColIdx(row, start+1)] = byte(fg) + '0'
			state.Chars[state.RowColIdx(row, start+2)] = byte(bg) + '0'
			style := MixStyle(styles[fg], styles[bg])
			for i := 0; i < 4; i++ {
				state.Styles[state.RowColIdx(row, start+i)] = style
			}
		}
	}
}

func displayByte(b byte) byte {
	assert(b != '\n')
	switch {
	case b >= 32 && b < 126:
		return b
	case b == '\t':
		return ' '
	default:
		return '?'
	}
}
