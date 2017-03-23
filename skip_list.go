package main

import "math/rand"

type skipList struct {
	header *element
}

type element struct {
	offset int
	data   string
	next   []*element
}

func newSkipList(levels int) skipList {
	assert(levels >= 1)
	return skipList{
		header: &element{
			next: make([]*element, levels),
		},
	}
}

func (s *skipList) insert(offset int, data string) {

	// TODO: Selecting height could be more efficient.
	height := 1
	for rand.Int()%2 == 0 && height+1 < len(s.header.next) {
		height++
	}

	newElement := &element{offset, data, make([]*element, height)}

	var fn func(*element, int)
	fn = func(root *element, level int) {

		if offset == root.offset {
			return
		}

		rightOfRoot := root.next[level]

		if rightOfRoot == nil || offset < rightOfRoot.offset {
			if level < height {
				root.next[level], newElement.next[level] = newElement, rightOfRoot
			}
			if level > 0 {
				fn(root, level-1)
			}
			return
		}

		fn(rightOfRoot, level)

	}
	fn(s.header, height)
}

func (s *skipList) find(offset int) *element {
	return nil
}

func (s *skipList) remove(count int) {
}
