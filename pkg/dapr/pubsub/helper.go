package pubsub

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/benc-uk/go-rest-api/pkg/problem"
	"github.com/go-chi/chi/v5"
)

type topic struct {
	PubSubName string `json:"pubSubName"`
	Topic      string `json:"topic"`
	Route      string `json:"route"`
}

type CloudEvent struct {
	ID   string      `json:"id"`
	Data interface{} `json:"data"`
}

const routeBase = "/dapr/pubsub/receive"

// Subscribe is a HTTP handler that lets Dapr know what topics we subscribe to
func Subscribe(pubSubName string, topics []string, router chi.Router) {
	log.Printf("### ✉️ DAPR: Subscribing to topics: %s", topics)

	router.Get("/dapr/subscribe", func(resp http.ResponseWriter, req *http.Request) {
		topicList := []topic{}
		for _, t := range topics {
			topicList = append(topicList, topic{
				PubSubName: pubSubName,
				Topic:      t,
				Route:      fmt.Sprintf("%s/%s", routeBase, t),
			})
		}
		json, _ := json.Marshal(topicList)

		resp.Header().Set("Content-Type", "application/json")
		_, _ = resp.Write(json)
	})
}

// AddTopicHandler is a way to plug in a handler for receiving messages from a Dapr pub-sub topic
// It's pretty simple and the handler is passed the CloudEvent data, for them to decode
func AddTopicHandler(topic string, router chi.Router, handler func(event *CloudEvent) error) {
	log.Printf("### ✉️ DAPR: Registered topic message handler: %s", topic)

	route := fmt.Sprintf("%s/%s", routeBase, topic)

	router.Post(route, func(resp http.ResponseWriter, req *http.Request) {
		// Decode the event
		event := &CloudEvent{}
		var bodyBytes []byte
		bodyBytes, _ = io.ReadAll(req.Body)

		// Basic validation checks
		err := json.Unmarshal(bodyBytes, &event)
		if err != nil {
			// Returning a non-200 will reschedule the received message
			problem.Wrap(500, req.RequestURI, topic, err).Send(resp)
		}

		// Log the event
		log.Printf("### 📩 Received message: %s from pub/sub topic: %s", topic, event.ID)

		// Pass the body to the handler
		// It would be really nice to pass the decoded data object/struct but we don't know the type
		err = handler(event)
		if err != nil {
			problem.Wrap(500, req.RequestURI, topic, err).Send(resp)
			return
		}
	})
}
