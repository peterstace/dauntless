package main

import "time"

const msgLingerDuration = 5 * time.Second

func setMessage(m *Model, msg string) {
	log.Info("Setting message: %q", msg)
	m.msg = msg
	m.msgSetAt = time.Now()
	go func() {
		// Redraw after message linger duration (message may not have to be
		// drawn anymore).
		time.Sleep(msgLingerDuration)
	}()
}
