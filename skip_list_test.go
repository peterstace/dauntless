package main

import "testing"

type content struct {
	offset int
	data   string
}

func SanityCheck(t *testing.T, s *skipList, contents []content) {

	var elements []*element
	for e := s.header.next[0]; e != nil; e = e.next[0] {
		elements = append(elements, e)
	}

	// Check the length
	{
		if len(elements) != len(contents) {
			t.Errorf("lens didn't match")
		}
	}

	// Check each element in the skip list was inserted.
	{
		for _, e := range elements {
			found := false
			for _, content := range contents {
				if content.offset == e.offset && content.data == e.data {
					found = true
				}
			}
			if !found {
				t.Errorf("Could not find matching node")
			}
		}
	}

	// Check elements are sorted.
	lastOffset := -1
	for _, e := range elements {
		lastOffsetValid := lastOffset >= 0
		if lastOffsetValid && lastOffset >= e.offset {
			t.Errorf("not sorted")
		}
	}

	// Check internal pointer consistency.
	{
		for level := range s.header.next {
			lastI := -1
			for e := s.header.next[level]; e != nil; e = e.next[level] {
				foundI := -1
				for i := range elements {
					if elements[i] == e {
						foundI = i
						break
					}
				}
				if foundI == -1 {
					t.Errorf("Could not find node")
				}
				if foundI <= lastI {
					t.Errorf("Node didn't advance")
				}
				lastI = foundI
			}
		}
	}

	// Check we can lookup each inserted piece of data by index.
	{
		for _, content := range contents {
			for i := range content.data {
				e := s.find(content.offset + i)
				if e == nil {
					t.Errorf("Expected lookup to succeed")
					continue
				}
				if e.offset != content.offset {
					t.Errorf("Wrong offset")
				}
				if e.data != content.data {
					t.Errorf("Wrong data")
				}
			}
		}
	}

	// Check prev links can traverse backwards.
	{
		if len(elements) > 0 {
			i := len(elements) - 1
			for e := elements[i]; e != s.header; e = e.prev {
				if e != elements[i] {
					t.Errorf("Prev link broken")
				}
				i--
			}
			if i != -1 {
				t.Errorf("Didn't get to the start of the list when following prev lists")
			}
		}
		if s.header.prev != nil {
			t.Errorf("expected header prev to be nil")
		}
	}

	// Make sure lookups of non-existing offsets fail.
	{
		minOffset := +9999
		maxOffset := -9999
		var testCount int
		for _, content := range contents {
			if content.offset < minOffset {
				minOffset = content.offset
			}
			if content.offset+len(content.data) > maxOffset {
				maxOffset = content.offset + len(content.data)
			}
		}
		minOffset--
		maxOffset++
		if minOffset < 0 {
			minOffset = 0
		}
		for i := minOffset; i < maxOffset; i++ {
			found := false
			for _, content := range contents {
				if i >= content.offset && i < content.offset+len(content.data) {
					found = true
				}
			}
			if found {
				continue
			}
			e := s.find(i)
			if e != nil {
				t.Errorf("Expected nil: %d", i)
			}
			testCount++
		}
		if len(contents) > 0 && testCount < 1 {
			t.Errorf("internal test inconsistency, should have at least "+
				"tested just before and just after max: %d", testCount, minOffset, maxOffset)
		}
	}
}

func TestSkipListInsert(t *testing.T) {
	for seed := 0; seed < 10; seed++ {
		for height := 1; height <= 3; height++ {
			for i, contents := range [][]content{
				{},
				{{0, "0123"}},
				{{0, "0123"}, {4, "4567"}},
				{{4, "4567"}, {0, "0123"}},
				{{0, "0123"}, {5, "5678"}},
				{{1, "1234"}},
			} {
				t.Logf("Seed=%d Height=%d Idx=%d", seed, height, i)
				s := newSkipList(height)
				for _, content := range contents {
					s.insert(content.offset, content.data)
				}
				SanityCheck(t, s, contents)
			}
		}
	}
}
