package main

import "testing"

func Expect(t *testing.T, b bool) {
	if !b {
		t.Fatal()
	}
}

type content struct {
	offset int
	data   string
}

func SanityCheck(t *testing.T, s *skipList, contents []content) {

	var elements []*element
	for e := s.header.next[0]; e != nil; e = e.next[0] {
		elements = append(elements, e)
	}

	i := 0
	for _, e := range elements {
		if contents[i].offset != e.offset {
			t.Errorf("Idx=%d WantOffset=%d GotOffset=%d", i, contents[i].offset, e.offset)
		}
		if contents[i].data != e.data {
			t.Errorf("Idx=%d WantData=%q GotData=%q", i, contents[i].data, e.data)
		}
		i++
	}

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

func TestSkipList_01_Empty(t *testing.T) {
	s := newSkipList(1)
	SanityCheck(t, s, []content{})
}

func TestSkipList_01_SingleInsert(t *testing.T) {

	s := newSkipList(1)
	s.insert(0, "0123")

	SanityCheck(t, s, []content{
		{0, "0123"},
	})
}

func TestSkipList_01_DoubleInsert(t *testing.T) {

	s := newSkipList(1)
	s.insert(0, "0123")
	s.insert(4, "4567")

	SanityCheck(t, s, []content{
		{0, "0123"},
		{4, "4567"},
	})
}

func TestSkipList_02_Empty(t *testing.T) {
	s := newSkipList(2)
	SanityCheck(t, s, []content{})
}

func TestSkipList_02_SingleInsert(t *testing.T) {

	s := newSkipList(2)
	s.insert(0, "0123")

	SanityCheck(t, s, []content{
		{0, "0123"},
	})
}

func TestSkipList_02_DoubleInsert(t *testing.T) {

	s := newSkipList(2)
	s.insert(0, "0123")
	s.insert(4, "4567")

	SanityCheck(t, s, []content{
		{0, "0123"},
		{4, "4567"},
	})
}
