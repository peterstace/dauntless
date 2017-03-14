package main

import "fmt"

func assert(b bool) {
	if !b {
		panic("assertion failed")
	}
}

type unloadedChunkError int

func (e unloadedChunkError) Error() string {
	return fmt.Sprintf("chunk %d is unloaded", int(e))
}

// Extracts lines from the chunks. If an error is returned, it is of type
// unloadedChunkError, and indicates the first unloaded chunk encountered. If
// not enough lines exist (even when the required chunks are loaded), all
// existing lines will still be returned (but the number of returned lines will
// be less than the number requested).
func extractLines(offset, numLines int, chunks map[int][]byte) ([]byte, error) {

	// Get the chunk that contains the current position.
	startChunkIdx := offset / chunkSize
	chunk, ok := chunks[startChunkIdx]
	if !ok {
		return nil, unloadedChunkError(startChunkIdx)
	}

	// Partial chunk at end of file. So the chunk contains all available lines.
	if len(chunk) < chunkSize {
		return chunk, nil
	}

	// Full chunk. Check to see if it contains the requested lines.
	assert(len(chunk) == chunkSize)
	newLineCount := 0
	enoughData := false
	for i := offset - startChunkIdx*chunkSize; i < len(chunk); i++ {
		if chunk[i] == '\n' {
			newLineCount++
			if newLineCount == numLines {
				enoughData = true
				break
			}
		}
	}
	if enoughData {
		return chunk, nil
	}

	// Requested lines spans multiple chunks. Build a new slice containing a
	// copy of the data. We have to do this because chunks may not be in
	// contiguous memory. Wasteful, but should be rare with large chunk size.
	newLineCount = 0
	i := offset
	var buf []byte
	for {
		chunkIdx := i / chunkSize
		chunk, ok := chunks[chunkIdx]
		if !ok {
			return nil, unloadedChunkError(chunkIdx)
		}
		inChunkIdx := i - chunkIdx*chunkSize
		if inChunkIdx >= len(chunk) {
			// End of file.
			return buf, nil
		}
		cell := chunk[inChunkIdx]
		buf = append(buf, cell)
		if cell == '\n' {
			newLineCount++
			if newLineCount == numLines {
				return buf, nil
			}
		}
		i++
	}

	assert(false)
	return nil, nil
}
