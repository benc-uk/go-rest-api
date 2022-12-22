// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020
// Licensed under the MIT License.
//
// Some example routes for the thing API
// ----------------------------------------------------------------------------

package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/benc-uk/go-rest-api/pkg/problem"
	"github.com/go-chi/chi/v5"
)

type ThingResp struct {
	Name string `json:"name"`
}

// All application routes should be registered here
func (api ThingAPI) addRoutes(router chi.Router) {
	router.Get("/things", api.getThings)
	router.Get("/things/{id}", api.getThingByID)
	router.Post("/things", api.createThing)
}

// Get all things, dummy implementation
func (api ThingAPI) getThings(resp http.ResponseWriter, req *http.Request) {
	things := make([]ThingResp, 0)

	things = append(things, ThingResp{
		Name: "Jimmy McGill",
	})
	things = append(things, ThingResp{
		Name: "Saul Goodman",
	})

	resp.Header().Set("Content-Type", "application/json")

	json, _ := json.Marshal(things)
	_, _ = resp.Write(json)
}

// Get a thing by ID, dummy implementation
func (api ThingAPI) getThingByID(resp http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")

	// Example of using problem package to send a 404
	if id != "1" {
		problem.Wrap(404, req.RequestURI, "thing", errors.New("thing not found")).Send(resp)
		return
	}

	thing := ThingResp{
		Name: "Jimmy McGill",
	}

	resp.Header().Set("Content-Type", "application/json")

	json, _ := json.Marshal(thing)
	_, _ = resp.Write(json)
}

// Create a new thing, dummy implementation
func (api ThingAPI) createThing(resp http.ResponseWriter, req *http.Request) {
	_, _ = resp.Write([]byte("OK"))
}
