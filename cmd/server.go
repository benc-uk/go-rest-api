// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020
// Licensed under the MIT License.
//
// Sample API server, using the go-rest-api package
// ----------------------------------------------------------------------------

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/benc-uk/go-rest-api/pkg/api"
	"github.com/benc-uk/go-rest-api/pkg/env"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"

	_ "github.com/joho/godotenv/autoload"
)

// ThingAPI is a wrap of the common base API with local implementation
type ThingAPI struct {
	*api.Base
	// Add extra fields here: database connections, SDK clients
}

var (
	healthy     = true               // Simple health flag
	version     = "0.0.1"            // App version number, set at build time with -ldflags "-X 'main.version=1.2.3'"
	buildInfo   = "No build details" // Build details, set at build time with -ldflags "-X 'main.buildInfo=Foo bar'"
	serviceName = "change-me"
	defaultPort = 8000
)

func main() {
	// Port to listen on, change the default as you see fit
	serverPort := env.GetEnvInt("PORT", defaultPort)

	router := chi.NewRouter()

	api := ThingAPI{
		api.NewBase(serviceName, version, buildInfo, healthy),
		// Database connections, SDK clients, etc can be added here
	}

	// Some basic middleware, change as you see fit, see: https://github.com/go-chi/chi#core-middlewares
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Some custom middleware for CORS & JWT username
	router.Use(api.SimpleCORSMiddleware)

	// *OPTIONAL* Add Prometheus metrics endpoint, must be before the other routes
	api.AddMetricsEndpoint(router, "metrics")

	// Add optional root, health & status endpoints
	api.AddHealthEndpoint(router, "health")
	api.AddStatusEndpoint(router, "status")
	api.AddOKEndpoint(router, "")

	// *OPTIONAL* Configure JWT validator with our token store and application scope
	// - Use chi router groups to add auth middleware to specific routes
	//jwtValidator := auth.NewJWTValidator("https://login.microsoftonline.com/common/discovery/v2.0/keys", "Some.Scope")
	//router.Use(jwtValidator.Middleware)

	// *OPTIONAL* Add support for single page applications (SPA) with client-side routing
	//log.Printf("### 🌏 Serving static files for SPA from: %s", "./")
	//router.Handle("/", static.SpaHandler{
	//	StaticPath: "./",
	//	IndexFile:  "index.html",
	//})

	// Main REST API routes for the application
	api.addRoutes(router)

	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", serverPort),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	log.Printf("### 🌐 %s API, listening on port: %d", serviceName, serverPort)
	log.Printf("### 🚀 Build details: v%s (%s)", version, buildInfo)
	log.Fatal(srv.ListenAndServe())
}
