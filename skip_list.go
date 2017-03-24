package main

import "math/rand"

type skipList struct {
	header *element
	rnd    *rand.Rand
}

type element struct {
	offset int
	data   string
	next   []*element
}

func newSkipList(levels int) *skipList {
	assert(levels >= 1)
	return &skipList{
		header: &element{
			offset: -1,
			data:   "__HEADER__",
			next:   make([]*element, levels),
		},
		rnd: rand.New(rand.NewSource(3)),
	}
}

func (s *skipList) insert(offset int, data string) {

	// TODO: Selecting height could be more efficient.
	height := 1
	for s.rnd.Int()%2 == 0 && height < len(s.header.next) {
		height++
	}

	newElement := &element{offset, data, make([]*element, height)}

	root := s.header
	level := height - 1
	for {

		assert(root != nil)

		// The line MUST NOT exist before being inserted. Before calling
		// insert, check to see if it exists using find (also check for any
		// data being changed at the same time).
		assert(root == s.header || offset >= root.offset+len(root.data))

		rightOfRoot := root.next[level]

		if rightOfRoot == nil || offset < rightOfRoot.offset {
			if level < height {
				root.next[level], newElement.next[level] = newElement, rightOfRoot
			}
			if level > 0 {
				level--
			} else {
				return
			}
		} else {
			root = rightOfRoot
		}
	}
}

func (s *skipList) find(offset int) *element {
	return nil
}
