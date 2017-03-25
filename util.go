package main

func assert(b bool) {
	if !b {
		panic("assertion failed")
	}
}
