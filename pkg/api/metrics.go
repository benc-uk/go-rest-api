package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

const unmatchedRoute = "unmatched"

type httpServerMetrics struct {
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

var (
	defaultHTTPServerMetricsOnce sync.Once
	defaultServerMetrics         *httpServerMetrics
)

func getDefaultHTTPServerMetrics() *httpServerMetrics {
	defaultHTTPServerMetricsOnce.Do(func() {
		defaultServerMetrics = newHTTPServerMetrics(prometheus.DefaultRegisterer)
	})

	return defaultServerMetrics
}

func newHTTPServerMetrics(registerer prometheus.Registerer) *httpServerMetrics {
	metrics := &httpServerMetrics{
		requestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_server_requests_total",
				Help: "Total number of HTTP server requests.",
			},
			[]string{"method", "route", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_server_request_duration_seconds",
				Help:    "Duration of HTTP server requests in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		),
	}

	registerer.MustRegister(metrics.requestCount, metrics.requestDuration)
	return metrics
}

func (metrics *httpServerMetrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		wrappedResponse := middleware.NewWrapResponseWriter(response, request.ProtoMajor)
		startedAt := time.Now()

		defer func() {
			recovered := recover()
			statusCode := wrappedResponse.Status()
			if statusCode == 0 {
				statusCode = http.StatusOK
				if recovered != nil {
					statusCode = http.StatusInternalServerError
				}
			}

			routePattern := unmatchedRoute
			if routeContext := chi.RouteContext(request.Context()); routeContext != nil {
				if matchedPattern := routeContext.RoutePattern(); matchedPattern != "" {
					routePattern = matchedPattern
				}
			}

			status := strconv.Itoa(statusCode)
			metrics.requestCount.WithLabelValues(request.Method, routePattern, status).Inc()
			duration := time.Since(startedAt).Seconds()
			metrics.requestDuration.WithLabelValues(request.Method, routePattern, status).Observe(duration)

			if recovered != nil {
				panic(recovered)
			}
		}()

		next.ServeHTTP(wrappedResponse, request)
	})
}
