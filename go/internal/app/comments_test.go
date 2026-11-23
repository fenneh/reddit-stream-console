package app

import "testing"

// — refresh lifecycle —

func TestStopAutoRefreshClosesChannel(t *testing.T) {
	ch := make(chan struct{})
	ta := &TviewApp{stopRefresh: ch}

	ta.stopAutoRefresh()

	select {
	case <-ch:
	default:
		t.Fatal("stopAutoRefresh did not close the refresh channel")
	}
	if ta.stopRefresh != nil {
		t.Error("stopRefresh not cleared")
	}
}

func TestStopAutoRefreshIsIdempotent(t *testing.T) {
	ta := &TviewApp{stopRefresh: make(chan struct{})}
	ta.stopAutoRefresh()
	ta.stopAutoRefresh()
}

func TestStartAutoRefreshCancelsPreviousLoop(t *testing.T) {
	ta := &TviewApp{}
	ta.startAutoRefresh()
	first := ta.stopRefresh

	ta.startAutoRefresh()

	select {
	case <-first:
	default:
		t.Fatal("starting a new refresh loop left the previous channel open")
	}
	ta.stopAutoRefresh()
}
