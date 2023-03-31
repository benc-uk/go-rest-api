// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020
// Licensed under the MIT License.
//
// Base API that all services implement and extend
// ----------------------------------------------------------------------------

package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/benc-uk/go-rest-api/pkg/problem"
	"github.com/go-chi/chi/v5"
)

// Base holds a standard set of values for all services & APIs
type Base struct {
	ServiceName string
	Healthy     bool
	Version     string
	BuildInfo   string
}

// NewBase creates and returns a new Base API instance
func NewBase(name, ver, info string, healthy bool) *Base {
	return &Base{
		ServiceName: name,
		Healthy:     healthy,
		Version:     ver,
		BuildInfo:   info,
	}
}

// ReturnJSON sends a JSON response to the client
func (b *Base) ReturnJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	dataBytes, err := json.Marshal(data)
	if err != nil {
		problem.Wrap(500, "json-encoding", "api-internals", err).Send(w)
		return
	}

	_, _ = w.Write(dataBytes)
}

// ReturnText sends a plain text response to the client
func (b *Base) ReturnText(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(msg))
}

// ReturnErrorJSON sends a JSON error response to the client
func (b *Base) ReturnErrorJSON(w http.ResponseWriter, err error) {
	b.ReturnJSON(w, map[string]string{"error": err.Error()})
}

// ReturnOKJSON sends a JSON OK response to the client
func (b *Base) ReturnOKJSON(w http.ResponseWriter) {
	b.ReturnJSON(w, map[string]string{"result": "ok"})
}

// StartServer starts the HTTP server and blocks until it exits
func (b *Base) StartServer(port int, router chi.Router, timeout time.Duration) {
	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", port),
		WriteTimeout: timeout,
		ReadTimeout:  timeout,
		IdleTimeout:  timeout,
	}

	log.Printf("### 🌐 %s API, listening on port: %d", b.ServiceName, port)
	log.Printf("### 🚀 Build details: %s (%s)", b.Version, b.BuildInfo)
	log.Fatal(srv.ListenAndServe())
}
