package assert

func True(b bool) {
	if !b {
		panic("assertion failed")
	}
}
