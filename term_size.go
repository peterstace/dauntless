package dauntless

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/peterstace/dauntless/term"
)

func CollectTermSize(r Reactor, a App) {
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGWINCH)
		sigCh <- nil // Prime so that we get a term size immediately.
		for {

			force := false
			select {
			case <-time.After(time.Second):
			case <-sigCh:
				force = true
			}

			rows, cols, err := term.GetTermSize()
			if err != nil {
				r.Stop(err)
				return
			}

			r.Enque(func() {
				if force {
					a.ForceRefresh()
				}
				a.TermSize(rows, cols)
			})
		}
	}()
}
