package dauntless

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestForwardLineReader(t *testing.T) {
	for _, test := range []struct {
		input string
		lines []string
	}{
		{"", nil},
		{"hello\n", []string{"hello\n"}},
		{"hello\nworld", []string{"hello\n"}},
		{"hello\nworld\n", []string{"hello\n", "world\n"}},
	} {
		reader := NewForwardLineReader(strings.NewReader(test.input), 0)
		reader.readBuf = make([]byte, 4) // Allow smaller test chunks.
		var got []string
		var err error
		for {
			var str string
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
			t.Fatalf("Unexpected error: %v", err)
		}
	}
}

func TestBackwardLineReaderSuccess(t *testing.T) {
	for _, test := range []struct {
		input string
		lines []string
	}{
		// Could add more test cases. But this is all we needed when we ran
		// into problems to help debugging.
		{"", nil},
		{"0", []string{"0"}},
		{"\n", []string{"\n"}},
		{"0123", []string{"0123"}},
		{"0123\n", []string{"0123\n"}},
		{"01234567890\n", []string{"01234567890\n"}},
	} {
		reader := NewBackwardLineReader(strings.NewReader(test.input), len(test.input))
		reader.readBuf = make([]byte, 4) // Allow smaller test chunks.
		var got []string
		var err error
		for {
			var str string
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
