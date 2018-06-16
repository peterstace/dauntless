package main

import (
	"fmt"
	"regexp"
	"runtime"
	"time"
)

func CreateView(m *Model) ScreenState {
	state := NewScreenState(m.rows, m.cols)
	state.Init()

	regexes := m.regexes
	if m.tmpRegex != nil {
		regexes = append(regexes, regex{MixStyle(Invert, Invert), m.tmpRegex})
	}
	if m.cmd.Mode == SearchCommand {
		if re, err := regexp.Compile(m.cmd.Text); err == nil {
			regexes = append(regexes, regex{MixStyle(Invert, Invert), re})
		}
	}

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
	if m.cmd.Mode != NoCommand {
		commandLineText = prompt(m.cmd.Mode) + m.cmd.Text
		state.ColPos = min(state.ColPos, len(prompt(m.cmd.Mode))+m.cmd.Pos)
	} else if m.longFileOpInProgress {
		commandLineText = "Long operation in progress (interrupt to cancel)"
	} else {
		if time.Now().Sub(m.msgSetAt) < msgLingerDuration {
			commandLineText = m.msg
		}
	}

	commandRow := m.rows - 1
	copy(state.Chars[commandRow*m.cols:(commandRow+1)*m.cols], commandLineText)
	if m.cmd.Mode == SearchCommand {
		if _, err := regexp.Compile(m.cmd.Text); err != nil {
			start := len(prompt(m.cmd.Mode))
			end := start + len(m.cmd.Text)
			for i := start; i < end; i++ {
				state.Styles[state.RowColIdx(commandRow, i)] = MixStyle(Red, Default)
			}
		}
	}

	if m.cmd.Mode == ColourCommand {
		overlaySwatch(state)
	}
	if m.debug {
		overlayDebug(m, state)
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

	var reStyle Style
	reLabel := "re"
	reStr := "<none>"
	if m.tmpRegex != nil {
		reLabel = "re(tmp)"
		reStr = m.tmpRegex.String()
		reStyle = MixStyle(Invert, Invert)
	} else if len(m.regexes) > 0 {
		reLabel = fmt.Sprintf("re(%d)", len(m.regexes))
		reStr = m.regexes[0].re.String()
		reStyle = m.regexes[0].style
	}

	statusRight := lineWrapMode + " " + pctStr + " "
	statusLeft := " " + m.filename + " " + reLabel + ":" + reStr

	for i := 0; i < len(reStr); i++ {
		offset := len(statusLeft) - len(reStr)
		state.Styles[statusRow*m.cols+offset+i] = reStyle
	}

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

func prompt(cmd CommandMode) string {
	switch cmd {
	case SearchCommand:
		return "Enter search regexp (interrupt to cancel): "
	case ColourCommand:
		return "Enter colour code (interrupt to cancel): "
	case SeekCommand:
		return "Enter seek percentage (interrupt to cancel): "
	case BisectCommand:
		return "Enter bisect target (interrupt to cancel): "
	case QuitCommand:
		return "Do you really want to quit? (y/n): "
	}
	assert(false)
	return ""
}

func overlayDebug(m *Model, state ScreenState) {

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	lines := []string{
		fmt.Sprintf("off: 0x%016x", m.offset),
		fmt.Sprintf("fwd: %d", len(m.fwd)),
		fmt.Sprintf("bck: %d", len(m.bck)),
		fmt.Sprintf("gomaxprocs: %d", runtime.GOMAXPROCS(0)),
		fmt.Sprintf("goroutines: %d", runtime.NumGoroutine()),
		fmt.Sprintf("numgc: %d", mem.NumGC),
	}

	var longestLength int
	for _, line := range lines {
		longestLength = max(longestLength, len(line))
	}

	startCol := state.Cols - longestLength - 4
	startRow := 1
	endCol := state.Cols - 2
	endRow := startRow + len(lines)

	for row := startRow; row < endRow; row++ {
		for col := startCol; col < endCol; col++ {
			idx := state.RowColIdx(row, col)
			state.Styles[idx] = MixStyle(Invert, Invert)
			state.Chars[idx] = ' '
		}
	}

	for i, line := range lines {
		copy(state.Chars[state.RowColIdx(i+startRow, startCol+1):], line)
	}
}
