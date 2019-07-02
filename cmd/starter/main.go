package main

//
// Basic REST API microservice, template/reference code
// Ben Coleman, July 2019, v1
//

import (
  "fmt"
  "net/http"
  "log"
  "os"
  "strings"

  "github.com/benc-uk/go-starter/pkg/envhelper"
  
  "github.com/gorilla/mux"
  _ "github.com/joho/godotenv/autoload" // Autoloads .env file if it exists
)

var (
  healthy   = true                // Simple health flag
  version   = "0.0.1"             // App version number, set at build time with -ldflags "-X main.version=1.2.3"
  buildInfo = "No build details"  // Build details, set at build time with -ldflags "-X main.buildInfo='Foo bar'"
)

//
// Main entry point, will start HTTP service
//
func main() {
  log.SetOutput(os.Stdout) // Personal preference on log output 
  log.Printf("### Starter microservice v%v starting...", version)

  // Port to listen on, change the default as you see fit
  serverPort := envhelper.GetEnvInt("PORT", 8000)  

  // Use gorilla/mux for routing  
  router := mux.NewRouter()     
  // Add middleware for logging and CORS
  router.Use(starterMiddleware) 

  // Application routes here
  router.HandleFunc("/healthz", routeHealthCheck)
  router.HandleFunc("/status", routeStatus)
  router.HandleFunc("/example", routeExample)

  // OPTIONAL - Remove this block or set to false as required
  // Handle static content if you want to host a SPA or other static HTML
  if enableStatic := true; enableStatic {
    staticDirectory := envhelper.GetEnvString("STATIC_DIR", ".")
    fileServer := http.FileServer(http.Dir(staticDirectory))
    router.PathPrefix("/js").Handler(http.StripPrefix("/", fileServer))
    router.PathPrefix("/css").Handler(http.StripPrefix("/", fileServer))
    router.PathPrefix("/img").Handler(http.StripPrefix("/", fileServer))

    // EVERYTHING else redirect to index.html
    router.NotFoundHandler = http.HandlerFunc(func (resp http.ResponseWriter, req *http.Request) {
      http.ServeFile(resp, req, staticDirectory + "/index.html")
    })

    log.Printf("### Serving static content from '%v'\n", staticDirectory)
  }

  // Start server
  log.Printf("### Server listening on %v\n", serverPort)
  err := http.ListenAndServe(fmt.Sprintf(":%d", serverPort), router)
  if err != nil {
		panic(err.Error())
  } 
}

//
// Log all HTTP requests with client address, method and request URI
// Plus a cheap and dirty CORS enabler
//
func starterMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
    resp.Header().Set("Access-Control-Allow-Origin", "*")
    log.Println("###", strings.Split(req.RemoteAddr, ":")[0], req.Method, req.RequestURI)
    next.ServeHTTP(resp, req)
  })
}