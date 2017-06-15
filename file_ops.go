package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"time"
)

func FindSeekOffset(filename string, seekPct float64) (int, error) {

	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return 0, err
	}

	offset := max(1, int(float64(fileInfo.Size())/100.0*seekPct))

	reader := NewBackwardLineReader(f, offset)
	line, err := reader.ReadLine()
	if err != nil {
		return 0, err
	}
	return offset - len(line), nil
}

func FindJumpToBottomOffset(filename string) (int, error) {

	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	size := int(info.Size())

	reader := NewBackwardLineReader(f, size)
	line, err := reader.ReadLine()
	if err == io.EOF {
		err = nil // Handles case where size is 0.
	}
	return size - len(line), err
}

func FindNextMatch(cancel *Cancellable, r Reactor, m *Model, start int, re *regexp.Regexp) {

	f, err := os.Open(m.filename)
	if err != nil {
		r.Stop(fmt.Errorf("Could not open file: %v", err))
		return
	}
	defer f.Close()

	time.Sleep(500 * time.Millisecond)

	if _, err := f.Seek(int64(start), 0); err != nil {
		r.Stop(fmt.Errorf("Could not seek: offset=%d", start))
		return
	}

	reader := bufio.NewReader(f)
	offset := start
	for {
		if cancel.Cancelled() {
			return
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				r.Stop(fmt.Errorf("Could not read: error=%v", err))
				return
			} else {
				r.Enque(func() {
					msg := "regex search complete: no match found"
					setMessage(m, msg)
				})
				return
			}
		}
		if re.Match(line) {
			break
		}
		offset += len(line)
	}

	r.Enque(func() {
		m.longFileOpInProgress = false
		log.Info("Regexp search completed with match.")
		moveToOffset(m, offset)
	})
}

func FindPrevMatch(cancel *Cancellable, r Reactor, m *Model, endOffset int, re *regexp.Regexp) {

	f, err := os.Open(m.filename)
	if err != nil {
		r.Stop(fmt.Errorf("Could not open file: %v", err))
		return
	}
	defer f.Close()

	time.Sleep(500 * time.Millisecond)

	lineReader := NewBackwardLineReader(f, endOffset)
	offset := endOffset
	for {
		if cancel.Cancelled() {
			return
		}

		line, err := lineReader.ReadLine()
		if err != nil {
			if err != io.EOF {
				r.Stop(fmt.Errorf("Could not read: error=%v", err))
				return
			} else {
				r.Enque(func() {
					msg := "regex search complete: no match found"
					setMessage(m, msg)
				})
				return
			}
		}
		offset -= len(line)
		if re.Match(line) {
			break
		}
	}

	r.Enque(func() {
		log.Info("Regexp search completed with match.")
		moveToOffset(m, offset)
	})
}

func Bisect(filename string, target string, mask *regexp.Regexp) (int, error) {

	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return 0, err
	}

	var start int
	end := int(fileInfo.Size() - 1)

	var i int
	for {
		i++
		if i == 1000 {
			return 0, errors.New("could not find target")
		}

		offset := start + rand.Intn(end-start+1)
		line, offset, err := lineAt(f, offset)
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
func lineAt(f *os.File, offset int) ([]byte, int, error) {

	if _, err := f.Seek(int64(offset), 0); err != nil {
		return nil, 0, err
	}
	fwdReader := bufio.NewReader(f)
	fwdBytes, err := fwdReader.ReadBytes('\n')
	if err != nil {
		return nil, 0, err
	}

	if _, err := f.Seek(int64(offset+len(fwdBytes)), 0); err != nil {
		return nil, 0, err
	}
	bckReader := NewBackwardLineReader(f, offset+len(fwdBytes))
	bckBytes, err := bckReader.ReadLine()
	return bckBytes, offset + len(fwdBytes) - len(bckBytes), err
}

func FindReloadOffset(filename string, offset int) (int, error) {

	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	_, n, err := lineAt(f, offset)
	if err == io.EOF {
		return FindJumpToBottomOffset(filename)
	}
	return n, err
}
