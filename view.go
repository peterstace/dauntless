package main

import (
	"errors"
	"fmt"
	"time"
)

func CreateView(m *Model) (ScreenState, error) {

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
			m.dataMissing = false
		} else if m.fileSize == 0 || len(m.fwd) != 0 && m.fwd[len(m.fwd)-1].nextOffset() >= m.fileSize {
			// Reached end of file. `m.fileSize` may be slightly out of date,
			// however next time it's updated the additional lines will be
			// displayed.
			state.Chars[state.RowColIdx(row, 0)] = '~'
			m.dataMissing = false
		} else if m.dataMissing && time.Now().Sub(m.dataMissingFrom) > loadingScreenGrace {
			// Haven't been able to display any data for at least the grace
			// period, so display the loading screen instead.
			buildLoadingScreen(state)
			break
		} else {
			// Cannot display the data, but within the grace period. Abort the
			// display procedure, trying again after the grace period.
			//
			// TODO: Should check the data *before* calling this function. This
			// function should not return an error.
			return ScreenState{}, errors.New("cannot display data")
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

	return state, nil
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

func buildLoadingScreen(state ScreenState) {
	state.Init() // Clear anything previously set.
	const loading = "Loading..."
	row := state.Rows() / 2
	startCol := (state.Cols - len(loading)) / 2
	copy(state.Chars[row*state.Cols+startCol:], loading)
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
