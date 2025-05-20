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
	"slices"
)

// Struct to hold the broker state
type Broker[T any] struct {
	// Push messages here to broadcast them to all connected clients
	//Broadcast chan T

	// New client connections, channel holds the clientID
	newClients chan string

	// Closed client connections, channel holds the clientID
	closingClients chan string

	// Main connections registry, keyed on clientID
	// Each client has their own message channel
	clients map[string]chan T

	// Map of client groups, keyed on group name
	groups map[string][]string

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

		newClients:     make(chan string),
		closingClients: make(chan string),
		clients:        make(map[string]chan T),
		groups:         make(map[string][]string),
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

	// Create a special group for all clients
	broker.groups["*"] = []string{}

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
			// Add this client to the special group for all clients
			broker.AddToGroup(clientID, "*")
			broker.ClientConnectedHandler(clientID)

		// CASE: Client has detached and we want to stop sending them messages
		case clientID := <-broker.closingClients:
			delete(broker.clients, clientID)

			// Remove client from all groups
			for group := range broker.groups {
				broker.RemoveFromGroup(clientID, group)
			}

			broker.ClientDisconnectedHandler(clientID)
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

// Send a message to a specific group of clients
func (broker *Broker[T]) SendToGroup(group string, message T) {
	for _, clientID := range broker.groups[group] {
		broker.clients[clientID] <- message
	}
}

// Send a message to all clients
func (broker *Broker[T]) SendToAll(message T) {
	for _, clientID := range broker.clients {
		clientID <- message
	}
}

// Add a client to a group
func (broker *Broker[T]) AddToGroup(clientID string, group string) {
	broker.groups[group] = append(broker.groups[group], clientID)
}

// Remove a client from a group
func (broker *Broker[T]) RemoveFromGroup(clientID string, group string) {
	broker.groups[group] = slices.DeleteFunc(broker.groups[group], func(cid string) bool {
		return cid == clientID
	})
}

// Get all groups
func (broker *Broker[T]) GetGroups() []string {
	var groups []string
	for group := range broker.groups {
		groups = append(groups, group)
	}

	return groups
}

// Get all clients in a group
func (broker *Broker[T]) GetGroupClients(group string) []string {
	return broker.groups[group]
}
