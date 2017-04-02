package main

// Assumes that the first byte is the start of a new line.
func extractLines(data []byte) []string {
	var lines []string
	startOfLine := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, string(data[startOfLine:i+1]))
			startOfLine = i + 1
		}
	}
	return lines
}
