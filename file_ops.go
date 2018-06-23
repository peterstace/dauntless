package dauntless

import (
	"errors"
	"io"
	"math/rand"
	"regexp"
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

func Bisect(content Content, target string, mask *regexp.Regexp) (int, error) {
	sz, err := content.Size()
	if err != nil {
		return 0, err
	}

	var start int
	end := int(sz - 1)

	var i int
	for {
		i++
		if i == 1000 {
			return 0, errors.New("could not find target")
		}

		offset := start + rand.Intn(end-start+1)
		line, offset, err := lineAt(content, offset)
		if err != nil {
			return 0, err
		}
		if start+len(line) >= end {
			break
		}
		if mask.Match(line) {
			if target < string(line) {
				end = offset
			} else {
				start = offset
			}
		}
	}

	return start, nil
}

// Gets the line containing the offset.
func lineAt(c Content, offset int) ([]byte, int, error) {
	fwdReader := NewForwardLineReader(c, offset)
	fwdBytes, err := fwdReader.ReadLine()
	if err != nil {
		return nil, 0, err
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
