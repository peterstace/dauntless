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
		line = eliminateOverStrike(line)
		lines = append(lines, string(line))
	}
	return lines, nil
}

func eliminateOverStrike(in []byte) []byte {
	out := make([]byte, 0, len(in))
	for i := 0; i < len(in); i++ {
		if i+2 < len(in) && in[i] == '_' && in[i+1] == '\b' {
			i += 2
		}
		out = append(out, in[i])
		if i+2 < len(in) && in[i+1] == '\b' && in[i] == in[i+2] {
			i += 2
		}
	}
	return out
}
