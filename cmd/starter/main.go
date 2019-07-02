package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/benc-uk/go-starter/pkg/envhelper"
	_ "github.com/joho/godotenv/autoload" // Autoload .env file
)

var contentDir = "."

func main() {
	fmt.Println(`### Smilr frontend service [Golang] starting...`);

	// Get server PORT setting or default
	serverPort := envhelper.GetEnvString("PORT", "3000")
	// Get CONTENT_DIR setting for static content or default

	if len(os.Args) > 1 {
		contentDir = os.Args[1]
	}

	// Routing
	muxrouter := mux.NewRouter()
	
	// Special config API route
	muxrouter.HandleFunc("/.config/{vars}", configRoute)

	// Handle static content, we have to explicitly put our top level dirs in here
	// - otherwise the NotFoundHandler will catch them
	fileServer := http.FileServer(http.Dir(contentDir))
	muxrouter.PathPrefix("/js").Handler(http.StripPrefix("/", fileServer))
	muxrouter.PathPrefix("/css").Handler(http.StripPrefix("/", fileServer))
	muxrouter.PathPrefix("/img").Handler(http.StripPrefix("/", fileServer))

	// EVERYTHING else redirect to index.html
	muxrouter.NotFoundHandler = http.HandlerFunc(spaIndexRoute) 

	// Extra info for debugging
	if apiEndpoint, exists := os.LookupEnv("API_ENDPOINT"); exists {
		fmt.Println("### Will use API endpoint:", apiEndpoint);
	}

	// Start server
	fmt.Printf("### Starting server listening on %v\n", serverPort)
	fmt.Printf("### Serving static content from '%v'\n", contentDir)
	http.ListenAndServe(":"+serverPort, muxrouter)
}

//
// Special route to handle serving static SPA content with a JS router
//
func spaIndexRoute(resp http.ResponseWriter, req *http.Request) {
	http.ServeFile(resp, req, contentDir + "/index.html")
}

//
// MICRO API allowing dynamic configuration of the client side Vue.js
// Allow caller to fetch a comma separated set of environmental vars from the server
//
func configRoute(resp http.ResponseWriter, req *http.Request) {
	data := make(map[string]string)

	varList := mux.Vars(req)["vars"]
	for _, key := range strings.Split(varList, ",") {
		data[key] = envhelper.GetEnvString(key, "")
	}

	json, _ := json.Marshal(data)
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Add("Content-Type", "application/json")
	resp.Write(json)
}