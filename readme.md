# Go - REST API Starter Kit & Library

This is a set of packages for creating REST & HTTP based microservices / backend servers in Go, with supporting functions and helpers. It's fairly opinionated and acts a little like a very minimal mini framework.

It purposefully doesn't use any web framework instead focusing on the base HTTP library and [Chi](https://github.com/go-chi/chi) for routing. Approaches such as composition which are idiomatic to Go, rather than classic dependency injection have been used.

The `cmd` folder has an example of a working server/service which will accept REST requests and has a minimal API, serving as a reference.

The `pkg` folder has a number of Go packages to support running REST APIs in Go, these are described in detail below

```
pkg/
├── api
├── auth
├── dapr/pubsub
├── env
├── httptester
├── problem
├── sse
└── static
```

This has been developed in Go 1.19+ and follows the https://github.com/golang-standards/project-layout guidelines for project structure. Which might be an acquired taste.

Also included are a standard and reusable Dockerfile & makefile, both of which inject version information into the build. The Makefile will also handle linting and running with hot-reload via `air`

## Package `api`

This is a baseline from which you can extend, in order to run your own API, see the `cmd/server.go` for an example of how this is done. A quick summary is:

```go
import "github.com/benc-uk/go-rest-api/pkg/api"

type MyAPI struct {
  // Embed and wrap the base API struct
  *api.Base

  // Add extra fields as per your implementation
  foo Foo
}

router := chi.NewRouter()
api := MyAPI{
  api.NewBase(serviceName, version, buildInfo, healthy),
}

api.AddHealthEndpoint(router, "health")
api.AddStatusEndpoint(router, "status")
```

The base API supports health checks and exposes data such as version and service name, plus helper functions for sending JSON & plain text responses or errors.

Optional endpoints which can be added to the API:

- Status endpoint, returning server & service details as JSON
- Prometheus metrics
- Health check
- Any routes you wish to return "200 OK" such as the root (/)

Optional middleware can be configured:

- Enabling permissive CORS policy (suggest you use chi/cors for finer grained control)
- Enriching HTTP request context with data extracted from JWT token. If a JWT token is found on any request, you can specify a claim, and the value of that claim will be extracted and put into the HTTP request context.

Supporting functions of the base API struct are, providing common API use cases:

```go
StartServer(port int, router chi.Router, timeout time.Duration)
ReturnJSON(w http.ResponseWriter, data interface{})
ReturnText(w http.ResponseWriter, msg string)
ReturnErrorJSON(w http.ResponseWriter, err error)
ReturnOKJSON(w http.ResponseWriter)
```

## Package `auth`

This package contains `Validator` interface which can be configured and used to enforce authentication on some or all routes of the API.

📝 Note: This package is generic and can be used with any code utilizing the `net/http` library

There are two implementations of the `Validator` interface:

- `PassthroughValidator` - Used when mocking & testing, or to conditionally switch auth off
- `JWTValidator` - Main JWT based validator

The `JWTValidator` takes three parameters when created:

- _Client ID_: An application client ID used when validating tokens, by checking the `aud` claim.
- _Scope_: A scope string, validated against the `scp` claim.
- _JWKS URL_: A URL of the keystore used to fetch public keys and and verify the signature of the token. This assumes tokens are signed with a public/private key algorithm e.g. RSA

It can be used two ways: `router.Use(jwtValidator.Middleware)` to add validating middleware to all routes on a router. Alternatively `jwtValidator.Protect(myHandler)` to wrap and protect certain handlers

Failed validation results in a HTTP 401 being returned.

## Package `env`

Very basic set of helpers for fetching env vars with fallbacks to default values.

## Package `problem`

Provides support for RFC-7807 standard formatted responses to API errors. Use the `Wrap()` function to wrap an error, and then `Send()` to write it to the HTTP response writer.

```go
// A rather contrived example
func (api MyAPI) getThing(resp http.ResponseWriter, req *http.Request) {
  id := "some_id"
  thing, err := api.dbContext.ExecuteSomeQuery(id, blah, blah)

  // Return a RFC-7807 problem wrapping the database error, HTTP 500 will be sent
  if err != nil {
    problem.Wrap(500, req.RequestURI, "thing:"+id, err).Send(resp)
    return
  }

  // Return a RFC-7807 problem describing the missing thing, HTTP 404 will be sent
  if thing == nil {
    problem.Wrap(404, req.RequestURI, "thing:"+id, errors.New("thing with that ID does not exist")).Send(resp)
    return
  }

  // Handle success
}
```

## Package `static`

Includes a `SpaHandler` for serving SPA style static applications which may contain client routing logic. It acts as a wrapper around the standard `http.FileServer` but rather than returning 404s it will return a fallback index file, e.g. _index.html_

Usage:

```go
r := chi.NewRouter()

r.Handle("/*", static.SpaHandler{
  StaticPath: "./",         // Path to app content directory
  IndexFile:  "index.html", // Name of your SPA HTML file
})

srv := &http.Server{
  Handler:      r,
  Addr:         ":8080",
}

log.Fatal(srv.ListenAndServe())
```

## Package `httptester`

Used to run through multiple test cases when integration testing an API or any HTTP service. Use the `httptester.TestCase` struct and pass an array of them to `httptester.Run()`

## Package `dapr/pubsub`

Use to register your API with Dapr pub-sub and subscribe to a given topic and register a callback handler for messages received at that topic.

## Package `logging`

Provides `FilteredRequestLogger` an extension of chi middleware logger which supports filtering out of requests from the logging output.

## Package `sse`

Provides support for [Server Side Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events). Two implementations are provided:

- Simple backend helper that can stream SSE events over HTTP, **WARNING!** _You probably don't want to ever use this as it supports only a single client at a time. It is here for completeness and to show how to implement SSE in Go._
- A message broker which can be used to send messages to multiple clients, groups and keep track of connections/disconnections

Note. This package is standalone and will work with any Go HTTP implementation, you don't need to be using the `api` or the other packages here.

Broker usage:

```go
srv := sse.NewBroker[string]()

// MessageAdapter is optional, but can (re)format messages
srv.MessageAdapter = func(message string, clientID string) sse.SSE {
  return sse.SSE{
    Event: "message",
    Data:  "Some prefix " + message,
  }
}

// Send a message every 1 second
go func() {
  for {
    srv.SendToAll("Hello! " + time.Now().String())
    time.Sleep(1 * time.Second)
  }
}()

http.HandleFunc("/stream-events", func(w http.ResponseWriter, r *http.Request) {
  clientID := "You need to implement something here"
  srv.Stream(clientID, w, *r)
})
```
