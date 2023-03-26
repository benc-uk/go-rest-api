// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2022
// Licensed under the MIT License.
//
// Optional extra endpoints you may want to add to your API
// ----------------------------------------------------------------------------

package api

import (
	"log"
	"net/http"
	"runtime"

	"github.com/elastic/go-sysinfo"
	"github.com/go-chi/chi/v5"
	metrics "github.com/m8as/go-chi-metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Status struct {
	Service      string `json:"service"`
	Healthy      bool   `json:"healthy"`
	Version      string `json:"version"`
	BuildInfo    string `json:"buildInfo"`
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	CPUCount     int    `json:"cpuCount"`
	GoVersion    string `json:"goVersion"`
	ClientAddr   string `json:"clientAddr"`
	ServerHost   string `json:"serverHost"`
	Uptime       string `json:"uptime"`
}

// AddOKEndpoint adds an endpoint that respond 200 when hitting it
func (b *Base) AddOKEndpoint(r chi.Router, path string) {
	log.Printf("### 🍏 API: 200 OK endpoint at: %s", "/"+path)

	r.Get("/"+path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b.ReturnText(w, "OK")
	})
}

// AddMetrics adds Prometheus metrics to the router
func (b *Base) AddMetricsEndpoint(r chi.Router, path string) {
	log.Printf("### 🔬 API: metrics endpoint at: %s", "/"+path)

	r.Use(metrics.SetRequestDuration)
	r.Use(metrics.IncRequestCount)
	r.Handle("/"+path, promhttp.Handler())
}

// AddHealth adds a health check endpoint to the API
func (b *Base) AddHealthEndpoint(r chi.Router, path string) {
	log.Printf("### 💚 API: health endpoint at: %s", "/"+path)

	r.HandleFunc("/"+path, func(w http.ResponseWriter, r *http.Request) {
		if b.Healthy {
			w.WriteHeader(http.StatusOK)
			b.ReturnText(w, "OK: Service is healthy")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			b.ReturnText(w, "Error: Service is not healthy")
		}
	})
}

// AddStatus adds a status & info endpoint to the API
func (b *Base) AddStatusEndpoint(r chi.Router, path string) {
	log.Printf("### 🔮 API: status endpoint at: %s", "/"+path)

	r.HandleFunc("/"+path, func(w http.ResponseWriter, r *http.Request) {
		host, _ := sysinfo.Host()
		host.Info().Uptime()

		status := Status{
			Service:      b.ServiceName,
			Healthy:      b.Healthy,
			Version:      b.Version,
			BuildInfo:    b.BuildInfo,
			Hostname:     host.Info().Hostname,
			GoVersion:    runtime.Version(),
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
			CPUCount:     runtime.NumCPU(),
			ClientAddr:   r.RemoteAddr,
			ServerHost:   r.Host,
		}

		b.ReturnJSON(w, status)
	})
}
