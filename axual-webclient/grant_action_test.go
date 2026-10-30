package webclient

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGrantActionsSendAtMostFourRequestsAtATime(t *testing.T) {
	var running, most atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := running.Add(1)
		for {
			m := most.Load()
			if n <= m || most.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		running.Add(-1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	client := &Client{HTTPClient: server.Client(), ApiURL: server.URL}

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.RevokeOrDenyGrant("g", "reason"); err != nil {
				t.Errorf("deny error = %v", err)
			}
		}()
	}
	wg.Wait()
	if got := most.Load(); got != maxGrantActions {
		t.Errorf("at most %d requests ran at the same time, expected %d", got, maxGrantActions)
	}
}
