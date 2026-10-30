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

// Every grant action waits for a free slot: with all slots taken, none reaches the server.
func TestApproveCancelAndDenyWaitForAFreeSlot(t *testing.T) {
	actions := map[string]func(c *Client) error{
		"approve": func(c *Client) error { return c.ApproveGrant("g") },
		"cancel":  func(c *Client) error { return c.CancelGrant("g") },
		"deny":    func(c *Client) error { return c.RevokeOrDenyGrant("g", "reason") },
	}
	for name, action := range actions {
		t.Run(name, func(t *testing.T) {
			hit := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hit <- struct{}{}
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)
			client := &Client{HTTPClient: server.Client(), ApiURL: server.URL}
			_ = client.grantAction(func() error { return nil }) // creates the slots
			for i := 0; i < maxGrantActions; i++ {              // take every slot
				client.grantActions <- struct{}{}
			}
			go func() { _ = action(client) }() // approve and cancel then sleep 10 s; the test doesn't wait for it

			select {
			case <-hit:
				t.Fatal("the request was sent while every slot was taken")
			case <-time.After(100 * time.Millisecond):
			}
			<-client.grantActions // free one slot
			select {
			case <-hit:
			case <-time.After(2 * time.Second):
				t.Fatal("the request wasn't sent after a slot was freed")
			}
		})
	}
}
