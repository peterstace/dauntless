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

func nextNewLine(p []byte) (int, bool) {
	for i, b := range p {
		if b == '\n' {
			return i, true
		}
	}
	return 0, false
}

func mustFindNewLine(p []byte) int {
	n, ok := nextNewLine(p)
	assert(ok)
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Extracts lines from the chunks. If an error is returned, it is of type
// unloadedChunkError, and indicates the first unloaded chunk encountered. If
// not enough lines exist (even when the required chunks are loaded), all
// existing lines will still be returned (but the number of returned lines will
// be less than the number requested). Doesn't return more than the requested
// number of lines.
func extractLines(offset, numLines int, chunks map[int][]byte) ([]byte, error) {

	// Get the chunk that contains the current position.
	startChunkIdx := offset / chunkSize
	chunk, ok := chunks[startChunkIdx]
	if !ok {
		return nil, unloadedChunkError(startChunkIdx)
	}

	// Partial chunk at end of file. So the chunk contains all available lines.
	if len(chunk) < chunkSize {
		endIdx := offset - startChunkIdx*chunkSize
		for i := 0; i < numLines; i++ {
			n, ok := nextNewLine(chunk[endIdx:])
			if !ok {
				// Chunk doesn't contain all of the requested lines, so return
				// a slice until the end of the chunk.
				return chunk[offset-startChunkIdx*chunkSize:], nil
			}
			endIdx += n
			endIdx++ // Advance past the newline.
		}
		return chunk[offset-startChunkIdx*chunkSize : endIdx], nil
	}

	// Full chunk. Check to see if it contains the requested lines.
	assert(len(chunk) == chunkSize)
	enoughData := true
	endIdx := offset - startChunkIdx*chunkSize
	for i := 0; i < numLines; i++ {
		n, ok := nextNewLine(chunk[endIdx:])
		if !ok {
			// Chunk doesn't contain all of the requested lines. There are
			// likely more lines in the following chunk. But there is a chance
			// that the file ends exactly on the chunk boundary.
			//
			// TODO: Handle the case where the file ends on the chunk boundary.
			enoughData = false
			break
		}
		endIdx += n
		endIdx++ // Advance past the newline.
	}
	if enoughData {
		return chunk[offset-startChunkIdx*chunkSize : endIdx], nil
	}

	// Requested lines spans multiple chunks. Build a new slice containing a
	// copy of the data. We have to do this because chunks may not be in
	// contiguous memory. Wasteful, but should be rare with large chunk size.
	newLineCount := 0
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
