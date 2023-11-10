// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2023
// Licensed under the MIT License.
//
// Generic SSE handler/helper for streaming events to a single client
// ----------------------------------------------------------------------------

package sse

import (
	"fmt"
	"net/http"
)

type Streamer[T any] struct {
	// Push messages here to send to the connected client
	Messages chan T

	// Handlers for client connection/disconnection
	ClientDisconnectedHandler func()

	// Message adapter, used to convert messages to SSE format
	// Expected that people will implement their own adapters for formatting and logic
	MessageAdapter func(message T) SSE
}

// Create a new Streamer
func NewStreamer[T any]() *Streamer[T] {
	srv := &Streamer[T]{
		// Buffered channel so we don't block
		Messages: make(chan T, 100),
	}

	// Default message adapter, just converts to a string
	srv.MessageAdapter = func(message T) SSE {
		return SSE{
			Event: "message",
			Data:  fmt.Sprintf("%v", message),
		}
	}

	// Placeholder handlers, do nothing by default
	srv.ClientDisconnectedHandler = func() {}

	return srv
}

// HTTP handler for connecting clients to the stream and sending SSE events
func (server *Streamer[T]) Stream(w http.ResponseWriter, r http.Request) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Listen to connection closing
	go func() {
		<-r.Context().Done()
		server.ClientDisconnectedHandler()
	}()

	defer func() {
		server.ClientDisconnectedHandler()
	}()

	// Main loop for sending messages to the client
	for {
		// Blocks here until there is a new message
		msg := <-server.Messages

		// Convert the message to SSE format via the adapter
		sse := server.MessageAdapter(msg)

		// Write and flush immediately as we are streaming data
		sse.Write(w)
		w.(http.Flusher).Flush()
	}
}
