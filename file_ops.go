package dauntless

import (
	"io"
)

func FindSeekOffset(c Content, seekPct float64) (int, error) {
	sz, err := c.Size()
	if err != nil {
		return 0, err
	}
	offset := max(1, int(float64(sz)/100.0*seekPct))
	reader := NewBackwardLineReader(c, offset)
	line, err := reader.ReadLine()
	if err != nil {
		return 0, err
	}
	return offset - len(line), nil
}

func FindJumpToBottomOffset(content Content) (int, error) {
	size, err := content.Size()
	if err != nil {
		return 0, err
	}
	reader := NewBackwardLineReader(content, int(size))
	line, err := reader.ReadLine()
	if err == io.EOF {
		err = nil // Handles case where size is 0.
	}
	return int(size) - len(line), err
}

// Gets the line containing the offset.
func lineAt(c Content, offset int) (string, int, error) {
	fwdReader := NewForwardLineReader(c, offset)
	fwdBytes, err := fwdReader.ReadLine()
	if err != nil {
		return "", 0, err
	}
	bckReader := NewBackwardLineReader(c, offset+len(fwdBytes))
	bckBytes, err := bckReader.ReadLine()
	return bckBytes, offset + len(fwdBytes) - len(bckBytes), err
}

func FindReloadOffset(content Content, offset int) (int, error) {
	_, n, err := lineAt(content, offset)
	if err == io.EOF {
		return FindJumpToBottomOffset(content)
	}
	return n, err
}
