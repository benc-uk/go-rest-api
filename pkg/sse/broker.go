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
	"sync"
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

	// Guards clients & groups, which are written by Stream & listen and read
	// by the send paths
	mu sync.RWMutex

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
	broker.mu.Lock()
	broker.clients[clientID] = messageChan
	broker.mu.Unlock()

	// Signal the broker that we have a new connection
	broker.newClients <- clientID

	// Remove this client from the map of connected clients, when this handler exits.
	defer func() {
		broker.closingClients <- clientID
	}()

	// Main loop for sending messages to the client, exits when the client
	// connection goes away
	for {
		select {
		case msg := <-messageChan:
			// Convert the message to SSE format via the adapter
			sse := broker.MessageAdapter(msg, clientID)

			// Write and flush immediately as we are streaming data
			sse.Write(w)
			w.(http.Flusher).Flush()

		case <-r.Context().Done():
			// Client has disconnected, stop streaming to it
			return nil
		}
	}
}

// Listen on different channels and act accordingly
func (broker *Broker[T]) listen() {
	for {
		select {
		// CASE: New client has connected
		case clientID := <-broker.newClients:
			broker.AddToGroup(clientID, "*")
			broker.ClientConnectedHandler(clientID)

		// CASE: Client has detached and we want to stop sending them messages
		case clientID := <-broker.closingClients:
			// Remove client from all groups, _before_ dropping the client's
			// channel, so senders never see a group member with no channel
			broker.RemoveFromAllGroups(clientID)

			broker.mu.Lock()
			delete(broker.clients, clientID)
			broker.mu.Unlock()

			broker.ClientDisconnectedHandler(clientID)
		}
	}
}

// Get all active clients in the broker
func (broker *Broker[T]) GetClients() []string {
	broker.mu.RLock()
	defer broker.mu.RUnlock()

	var clients []string
	for clientID := range broker.clients {
		clients = append(clients, clientID)
	}

	return clients
}

// Get the number of active clients in the broker
func (broker *Broker[T]) GetClientCount() int {
	broker.mu.RLock()
	defer broker.mu.RUnlock()

	return len(broker.clients)
}

// Send a message to a specific client rather than broadcasting
// Never blocks: clients that are not connected to this broker, or that are
// too slow to keep up, are skipped
func (broker *Broker[T]) SendToClient(clientID string, message T) {
	broker.mu.RLock()
	ch, ok := broker.clients[clientID]
	broker.mu.RUnlock()

	if !ok {
		return
	}

	select {
	case ch <- message:
	default:
		// Client is not ready to receive, drop the message rather than block
	}
}

// Send a message to a specific group of clients
// Never blocks: clients that are not connected to this broker, or that are
// too slow to keep up, are skipped
func (broker *Broker[T]) SendToGroup(group string, message T) {
	// Snapshot the channels under lock, so we don't hold the lock while sending
	broker.mu.RLock()
	channels := make([]chan T, 0, len(broker.groups[group]))
	for _, clientID := range broker.groups[group] {
		if ch, ok := broker.clients[clientID]; ok {
			channels = append(channels, ch)
		}
	}
	broker.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- message:
		default:
			// Client is not ready to receive, drop the message rather than block
		}
	}
}

// Send a message to all clients
// Never blocks: clients that are too slow to keep up are skipped
func (broker *Broker[T]) SendToAll(message T) {
	broker.mu.RLock()
	channels := make([]chan T, 0, len(broker.clients))
	for _, ch := range broker.clients {
		channels = append(channels, ch)
	}
	broker.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- message:
		default:
			// Client is not ready to receive, drop the message rather than block
		}
	}
}

// Add a client to a group
func (broker *Broker[T]) AddToGroup(clientID string, group string) {
	broker.mu.Lock()
	defer broker.mu.Unlock()

	broker.groups[group] = append(broker.groups[group], clientID)
}

// Remove a client from a group
func (broker *Broker[T]) RemoveFromGroup(clientID string, group string) {
	broker.mu.Lock()
	defer broker.mu.Unlock()

	broker.groups[group] = slices.DeleteFunc(broker.groups[group], func(cid string) bool {
		return cid == clientID
	})
}

// Remove from all groups
func (broker *Broker[T]) RemoveFromAllGroups(clientID string) {
	broker.mu.Lock()
	defer broker.mu.Unlock()

	for group := range broker.groups {
		broker.groups[group] = slices.DeleteFunc(broker.groups[group], func(cid string) bool {
			return cid == clientID
		})
	}
}

// Get all groups
func (broker *Broker[T]) GetGroups() []string {
	broker.mu.RLock()
	defer broker.mu.RUnlock()

	var groups []string
	for group := range broker.groups {
		groups = append(groups, group)
	}

	return groups
}

// Get all clients in a group
func (broker *Broker[T]) GetGroupClients(group string) []string {
	broker.mu.RLock()
	defer broker.mu.RUnlock()

	return broker.groups[group]
}
