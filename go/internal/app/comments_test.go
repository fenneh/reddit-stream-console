package app

import (
	"testing"
	"time"
)

// — stopAutoRefresh —

func TestStopAutoRefreshClearsEnabledFlag(t *testing.T) {
	ta := &TviewApp{refreshEnabled: true, stopRefresh: make(chan struct{})}
	ta.stopAutoRefresh()
	if ta.refreshEnabled {
		t.Error("refreshEnabled not cleared")
	}
}

func TestStopAutoRefreshDoesNotBlockWithoutListener(t *testing.T) {
	ta := &TviewApp{refreshEnabled: true, stopRefresh: make(chan struct{})}

	done := make(chan struct{})
	go func() {
		ta.stopAutoRefresh()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stopAutoRefresh blocked with no listener on stopRefresh")
	}
}

func TestStopAutoRefreshSignalsListener(t *testing.T) {
	ta := &TviewApp{refreshEnabled: true, stopRefresh: make(chan struct{})}

	received := make(chan struct{})
	go func() {
		<-ta.stopRefresh
		close(received)
	}()

	// stopAutoRefresh's send is a non-blocking select, so it can race the
	// listener goroutine above onto its receive; retry until it lands.
	deadline := time.After(time.Second)
	for {
		select {
		case <-received:
			return
		case <-deadline:
			t.Fatal("listener never received stop signal")
		default:
			ta.stopAutoRefresh()
			time.Sleep(time.Millisecond)
		}
	}
}
