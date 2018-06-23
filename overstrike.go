package main

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
