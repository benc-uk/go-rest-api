// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2023
// Licensed under the MIT License.
//
// Tests for the SSE broker, focused on the guarantee that senders can never
// block, even when group members have no connection on this broker (e.g.
// multi-replica deployments) or when a client is too slow to keep up
// ----------------------------------------------------------------------------

package sse

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// how long to wait before declaring that a send blocked, which is the bug
// these tests guard against
const blockTimeout = 1 * time.Second

// threadSafeRecorder is an http.ResponseWriter that Stream can write to while
// the test reads the body without a data race
type threadSafeRecorder struct {
	mu   sync.Mutex
	body bytes.Buffer
}

func (r *threadSafeRecorder) Header() http.Header    { return http.Header{} }
func (r *threadSafeRecorder) WriteHeader(status int) {}
func (r *threadSafeRecorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.Write(b)
}
func (r *threadSafeRecorder) Flush() {}
func (r *threadSafeRecorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.String()
}

// connectClient runs Stream in the background and returns once the broker has
// registered the client, plus functions to read the stream and end it
func connectClient(t *testing.T, broker *Broker[string], clientID string) (body func() string, end func()) {
	t.Helper()

	connected := make(chan struct{})
	broker.ClientConnectedHandler = func(id string) {
		close(connected)
	}

	recorder := &threadSafeRecorder{}
	ctx, cancel := context.WithCancel(context.Background())
	streamDone := make(chan error, 1)
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/updates", nil).WithContext(ctx)
		streamDone <- broker.Stream(clientID, recorder, *req)
	}()

	select {
	case <-connected:
	case <-time.After(blockTimeout):
		t.Fatal("client did not connect in time")
	}

	return recorder.String, cancel
}

// waitFor polls until cond is true or the timeout hits, used instead of sleeps
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(blockTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met in time")
}

func TestSendToGroupSkipsGroupMemberWithoutConnection(t *testing.T) {
	broker := NewBroker[string]()

	// The bug: a group member whose SSE stream lives on another replica, so
	// the broker has no channel for it
	broker.AddToGroup("phantom-client", "ns1")
	broker.AddToGroup("also-phantom", "ns1")

	done := make(chan struct{})
	go func() {
		broker.SendToGroup("ns1", "hello")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(blockTimeout):
		t.Fatal("SendToGroup blocked on a group member with no connection")
	}
}

func TestSendToClientSkipsClientWithoutConnection(t *testing.T) {
	broker := NewBroker[string]()

	done := make(chan struct{})
	go func() {
		broker.SendToClient("phantom-client", "hello")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(blockTimeout):
		t.Fatal("SendToClient blocked on a client with no connection")
	}
}

func TestSendToGroupDeliversToConnectedClient(t *testing.T) {
	broker := NewBroker[string]()

	body, end := connectClient(t, broker, "real-client")
	defer end()

	broker.AddToGroup("real-client", "ns1")

	// The send may race with the stream goroutine entering its receive loop,
	// in which case the message is dropped by design, so retry until it lands
	deadline := time.Now().Add(blockTimeout)
	for time.Now().Before(deadline) {
		broker.SendToGroup("ns1", "hello")
		if bytes.Contains([]byte(body()), []byte("hello")) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("message was not delivered to a connected client")
}

func TestSendToGroupDoesNotBlockOnSlowClient(t *testing.T) {
	broker := NewBroker[string]()

	// Connect a client, then stall its stream so it stops reading from its
	// message channel, simulating a slow or hung connection
	connected := make(chan struct{})
	broker.ClientConnectedHandler = func(id string) { close(connected) }

	stalled := &threadSafeRecorder{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/updates", nil).WithContext(ctx)
		_ = broker.Stream("slow-client", stalled, *req)
	}()

	select {
	case <-connected:
	case <-time.After(blockTimeout):
		t.Fatal("client did not connect in time")
	}

	// Fill any capacity and keep the stream busy by sending a burst without
	// letting the stalled stream consume anything - sends must still return
	for i := 0; i < 100; i++ {
		done := make(chan struct{})
		go func() {
			broker.SendToGroup("*", fmt.Sprintf("message %d", i))
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(blockTimeout):
			t.Fatalf("SendToGroup blocked on message %d with a slow client", i)
		}
	}
}

func TestStreamExitsWhenClientDisconnects(t *testing.T) {
	broker := NewBroker[string]()

	body, end := connectClient(t, broker, "leaky-client")
	end() // cancels the request context, i.e. the client closes the connection

	// The stream goroutine must exit rather than block on its message channel
	// forever, otherwise every client connection leaks a goroutine
	waitFor(t, func() bool {
		return len(broker.GetClients()) == 0
	})

	// And the client must have been removed from the groups it joined
	if len(broker.GetGroupClients("*")) != 0 {
		t.Fatalf("disconnected client still in group '*': %v", broker.GetGroupClients("*"))
	}

	_ = body()
}

func TestSendToAllDeliversAndNeverBlocks(t *testing.T) {
	broker := NewBroker[string]()

	body, end := connectClient(t, broker, "real-client")
	defer end()

	broker.SendToAll("broadcast")

	// Retry until the message lands, see TestSendToGroupDeliversToConnectedClient
	deadline := time.Now().Add(blockTimeout)
	for time.Now().Before(deadline) {
		broker.SendToAll("broadcast")
		if bytes.Contains([]byte(body()), []byte("broadcast")) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("broadcast was not delivered to a connected client")
}
