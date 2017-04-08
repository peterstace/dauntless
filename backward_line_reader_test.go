package main

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestBackwardLineReaderSuccess(t *testing.T) {

	for _, test := range []struct {
		input string
		lines []string
	}{
		// Could add more test cases. But this is all we needed when we ran
		// into problems to help debugging.
		{"", nil},
		{"0123", []string{"0123"}},
		{"0123\n", []string{"0123\n"}},
		{"01234567890\n", []string{"01234567890\n"}},
	} {
		reader := NewBackwardLineReader(strings.NewReader(test.input), len(test.input))
		var got []string
		var err error
		for {
			var str []byte
			str, err = reader.ReadLine()
			if err == nil {
				got = append(got, string(str))
				continue
			}
			break
		}
		if err == io.EOF {
			if !reflect.DeepEqual(got, test.lines) {
				t.Errorf("Got=%v Want=%v", got, test.lines)
			}
		} else {
			t.Errorf("Unexpected error: %v", err)
		}
	}

}
