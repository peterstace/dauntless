package main

import "testing"

func TestFindStartOfNextLineSuccess(t *testing.T) {
	for i, test := range []struct {
		offset     int
		firstChunk string
		want       int
	}{
		{
			0,
			"\n1\n3\n",
			1,
		},
		{
			0,
			"\n123\n",
			1,
		},
		{
			0,
			"\n\n234\n",
			1,
		},
		{
			0,
			"package main\n\nimport (\n\t\"flag\"\n",
			13,
		},
		{
			13,
			"package main\n\nimport (\n\t\"flag\"\n",
			14,
		},
		{
			0,
			"0123456\n" + "8901234\n",
			8,
		},
	} {
		chunks := map[int][]byte{0: []byte(test.firstChunk)}
		got, ok := findStartOfNextLine(test.offset, chunks)
		if !ok {
			t.Errorf("couldn't find next line")
		}
		if got != test.want {
			t.Errorf("%d: Got=%d Want=%d", i, got, test.want)
		}
	}
}
