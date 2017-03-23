package main

import "testing"

func TestSkipList_SingleInsert(t *testing.T) {

	s := newSkipList(1)
	s.insert(0, "0123")

	e := s.header.next[0]
	if !(e != nil) {
		t.Fatal()
	}
	if !(len(e.next) == 1) {
		t.Fatal()
	}
	if !(e.next[0] == nil) {
		t.Fatal()
	}
	if !(e.offset == 0) {
		t.Fatal()
	}
	if !(e.data == "0123") {
		t.Fatal()
	}
}

func TestSkipList_DoubleInsert(t *testing.T) {

	s := newSkipList(1)
	s.insert(0, "0123")
	s.insert(4, "4567")

	e1 := s.header.next[0]
	if !(e1 != nil) {
		t.Fatal()
	}
	if !(len(e1.next) == 1) {
		t.Fatal()
	}

	e2 := e1.next[0]
	if !(e2 != nil) {
		t.Fatal()
	}
	if !(len(e2.next) == 1) {
		t.Fatal()
	}

	if !(e2.next[0] == nil) {
		t.Fatal()
	}

	if !(e1.offset == 0) {
		t.Fatal()
	}
	if !(e1.data == "0123") {
		t.Fatal()
	}
	if !(e2.offset == 4) {
		t.Fatal()
	}
	if !(e2.data == "4567") {
		t.Fatal()
	}
}

func TestSkipList_InsertSame(t *testing.T) {

	s := newSkipList(1)
	s.insert(0, "0123")
	s.insert(0, "0123")

	e := s.header.next[0]
	if !(e != nil) {
		t.Fatal()
	}
	if !(len(e.next) == 1) {
		t.Fatal()
	}
	if !(e.next[0] == nil) {
		t.Fatal()
	}
	if !(e.offset == 0) {
		t.Fatal()
	}
	if !(e.data == "0123") {
		t.Fatal()
	}
}
