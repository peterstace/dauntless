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

	// TODO: Selecting level could be more efficient.
	height := 1
	for rand.Int()%2 == 0 && height-1 < len(s.header.next) {
		height++
	}

	newElement := &element{offset, data, make([]*element, height)}

	var fn func(*element, int)
	fn = func(root *element, currentLevel int) {

		rightOfRoot := root.next[currentLevel]
		if offset >= rightOfRoot.offset+len(rightOfRoot.data) {
			fn(rightOfRoot, currentLevel)
			return
		}

		if currentLevel == 0 {
			if offset == root.offset {
				return
			}
			tmp := rightOfRoot
			root.next[0] = newElement
			newElement.next[0] = tmp
			return
		}

		if currentLevel < height {
			tmp := rightOfRoot
			root.next[currentLevel] = newElement
			newElement.next[currentLevel] = tmp
			return
		}

		fn(root, currentLevel-1)
		return
	}
	fn(s.header, height)
}

func (s *skipList) find(offset int) *element {
	return nil
}

func (s *skipList) remove(count int) {
}
