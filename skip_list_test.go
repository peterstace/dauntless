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

func TestSkipListInsert(t *testing.T) {
	for seed := 0; seed < 10; seed++ {
		for height := 1; height <= 3; height++ {
			for i, contents := range [][]content{
				{},
				{{0, "0123"}},
				{{0, "0123"}, {4, "4567"}},
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
