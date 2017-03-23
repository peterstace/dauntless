package main

import "testing"

func TestSkipList_SingleInsert(t *testing.T) {
	s := newSkipList(1)
	s.insert(0, "0123")
}

func TestSkipList_DoubleInsert(t *testing.T) {
	s := newSkipList(1)
	s.insert(0, "0123")
	s.insert(4, "4567")
}

func TestSkipList_InsertSame(t *testing.T) {
	s := newSkipList(1)
	s.insert(0, "0123")
	s.insert(0, "0123")
}
