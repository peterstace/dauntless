package dauntless

import "io"

func LoadFwd(content Content, offset int, count int) ([]string, error) {
	r := NewForwardLineReader(content, offset)
	return load(count, r)
}

func LoadBck(content Content, offset int, count int) ([]string, error) {
	r := NewBackwardLineReader(content, offset)
	return load(count, r)
}

func load(count int, r LineReader) ([]string, error) {
	lines := make([]string, 0, count)
	for i := 0; i < count; i++ {
		line, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				return lines, nil
			} else {
				return nil, err
			}
		}
		lines = append(lines, string(line))
	}
	return lines, nil
}
