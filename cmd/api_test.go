// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020
// Licensed under the MIT License.
//
// Example set of tests
// ----------------------------------------------------------------------------

package main

import (
	"io"
	"log"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/benc-uk/go-rest-api/pkg/api"
	"github.com/benc-uk/go-rest-api/pkg/auth"
	"github.com/benc-uk/go-rest-api/pkg/httptester"
)

func TestUsers(t *testing.T) {
	// Comment out to see logs
	log.SetOutput(io.Discard)

	router := chi.NewRouter()
	api := ThingAPI{
		api.NewBase("thing", "ignore", "ignore", true),
		// inject mocks here
	}

	// Add optional endpoints
	api.AddOKEndpoint(router, "")

	// Test the protected routes and JWT validation
	router.Group(func(protectedRouter chi.Router) {
		jwtValidator := auth.NewJWTValidator("ignored", "https://change_me/jwks_endpoint", "ignored")
		protectedRouter.Use(jwtValidator.Middleware)
		api.addProtectedRoutes(protectedRouter)
	})

	api.addPublicRoutes(router)

	httptester.Run(t, router, testCases)
}

var testCases = []httptester.TestCase{
	{
		Name:           "get root URL",
		URL:            "/",
		Method:         "GET",
		Body:           ``,
		CheckBody:      "OK",
		CheckBodyCount: 1,
		CheckStatus:    200,
	},
	{
		Name:           "get things API",
		URL:            "/things",
		Method:         "GET",
		Body:           ``,
		CheckBody:      "Cheese",
		CheckBodyCount: 1,
		CheckStatus:    200,
	},
	{
		Name:           "post things API",
		URL:            "/things",
		Method:         "POST",
		Body:           `{"name":"Cheese"}`,
		CheckBody:      `{"result":"ok"}`,
		CheckBodyCount: 1,
		CheckStatus:    200,
	},
	{
		Name:           "invalid method",
		URL:            "/things",
		Method:         "PUT",
		Body:           ``,
		CheckBody:      ``,
		CheckBodyCount: 0,
		CheckStatus:    405,
	},
	{
		Name:           "invalid URL",
		URL:            "/goats",
		Method:         "GET",
		Body:           ``,
		CheckBody:      ``,
		CheckBodyCount: 0,
		CheckStatus:    404,
	},
	{
		Name:           "delete thing",
		URL:            "/things/1",
		Method:         "DELETE",
		Body:           ``,
		CheckBody:      ``,
		CheckBodyCount: 0,
		CheckStatus:    401,
	},
}
