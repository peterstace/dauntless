package main

func moveToOffset(m *Model, offset int) {
	log.Info("Moving to offset: currentOffset=%d newOffset=%d", m.offset, offset)

	assert(offset >= 0)

	if m.offset == offset {
		log.Info("Already at target offset.")
	} else if offset < m.offset {
		moveUpToOffset(m, offset)
	} else {
		moveDownToOffset(m, offset)
	}
}

func moveUpToOffset(m *Model, offset int) {

	log.Info("Moving up to offset: currentOffset=%d newOffset=%d", m.offset, offset)

	haveTargetLoaded := false
	for _, ln := range m.bck {
		if ln.offset == offset {
			haveTargetLoaded = true
			break
		}
	}
	if haveTargetLoaded {
		for m.offset != offset {
			ln := m.bck[0]
			m.fwd = append([]line{ln}, m.fwd...)
			m.bck = m.bck[1:]
			m.offset = ln.offset
		}
	} else {
		m.fwd = nil
		m.bck = nil
		m.offset = offset
	}
}

func moveDownToOffset(m *Model, offset int) {

	log.Info("Moving down to offset: currentOffset=%d newOffset=%d", m.offset, offset)

	haveTargetLoaded := false
	for _, ln := range m.fwd {
		if ln.offset == offset {
			haveTargetLoaded = true
			break
		}
	}
	if haveTargetLoaded {
		for m.offset != offset {
			ln := m.fwd[0]
			m.fwd = m.fwd[1:]
			m.bck = append([]line{ln}, m.bck...)
			m.offset = ln.offset + len(ln.data)
		}
	} else {
		m.fwd = nil
		m.bck = nil
		m.offset = offset
	}
}
