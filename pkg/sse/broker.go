// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2023
// Licensed under the MIT License.
//
// Generic SSE message broker handles multiple SSE connections and events
// ----------------------------------------------------------------------------

package sse

import (
	"fmt"
	"net/http"
)

// Struct to hold the broker state
type Broker[T any] struct {
	// Push messages here to broadcast them to all connected clients
	Broadcast chan T

	// New client connections, channel holds the clientID
	newClients chan string

	// Closed client connections, channel holds the clientID
	closingClients chan string

	// Main connections registry, keyed on clientID
	// Each client has their own message channel
	clients map[string]chan T

	// Handlers for client connection/disconnection
	ClientConnectedHandler    func(clientID string)
	ClientDisconnectedHandler func(clientID string)

	// Message adapter, used to convert messages to SSE format
	// Expected that people will implement their own adapters for formatting and logic
	MessageAdapter func(message T, clientID string) SSE
}

// Create a new broker
func NewBroker[T any]() *Broker[T] {
	broker := &Broker[T]{
		// Buffered channel so we don't block
		Broadcast:      make(chan T, 100),
		newClients:     make(chan string),
		closingClients: make(chan string),
		clients:        make(map[string]chan T),
	}

	// Default message adapter, just converts to a string
	broker.MessageAdapter = func(message T, clientID string) SSE {
		return SSE{
			Event: "message",
			Data:  fmt.Sprintf("%v", message),
		}
	}

	// Placeholder handlers, do nothing by default
	broker.ClientConnectedHandler = func(clientID string) {}
	broker.ClientDisconnectedHandler = func(clientID string) {}

	// Set it running, listening and broadcasting events
	// Note: This runs in a goroutine so we don't block here
	go broker.listen()

	return broker
}

// HTTP handler for connecting clients to the stream and sending SSE events
func (broker *Broker[T]) Stream(clientID string, w http.ResponseWriter, r http.Request) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the broker's connections registry
	messageChan := make(chan T)
	broker.clients[clientID] = messageChan

	// Signal the broker that we have a new connection
	broker.newClients <- clientID

	// Remove this client from the map of connected clients, when this handler exits.
	defer func() {
		broker.closingClients <- clientID
	}()

	// Listen to connection closing and un-register client
	go func() {
		<-r.Context().Done()
		broker.closingClients <- clientID
	}()

	// Main loop for sending messages to the client
	for {
		// Blocks here until there is a new message in this client's messageChan
		msg := <-messageChan

		// Convert the message to SSE format via the adapter
		sse := broker.MessageAdapter(msg, clientID)

		// Write and flush immediately as we are streaming data
		sse.Write(w)
		w.(http.Flusher).Flush()
	}
}

// Listen on different channels and act accordingly
func (broker *Broker[T]) listen() {
	for {
		select {
		// CASE: New client has connected
		case clientID := <-broker.newClients:
			broker.ClientConnectedHandler(clientID)

		// CASE: Client has detached and we want to stop sending them messages
		case clientID := <-broker.closingClients:
			delete(broker.clients, clientID)
			broker.ClientDisconnectedHandler(clientID)

		// CASE: Message incoming on the broadcast channel
		case message := <-broker.Broadcast:
			// Loop through all connected clients and broadcast the message to their message channel
			for clientID := range broker.clients {
				broker.clients[clientID] <- message
			}
		}
	}
}

// Get all active clients in the broker
func (broker *Broker[T]) GetClients() []string {
	var clients []string
	for clientID := range broker.clients {
		clients = append(clients, clientID)
	}

	return clients
}

// Get the number of active clients in the broker
func (broker *Broker[T]) GetClientCount() int {
	return len(broker.clients)
}

// Send a message to a specific client rather than broadcasting
func (broker *Broker[T]) SendToClient(clientID string, message T) {
	broker.clients[clientID] <- message
}
