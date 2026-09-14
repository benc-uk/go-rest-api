package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

func TestHTTPServerMetricsUsesRoutePattern(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := newHTTPServerMetrics(registry)
	router := chi.NewRouter()
	router.Use(metrics.middleware)
	router.Get("/things/{thingID}", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})

	for _, target := range []string{"/things/123?view=full", "/things/456?view=summary"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
		}
	}

	assertRequestCountMetric(t, registry)
	assertRequestDurationMetric(t, registry)
}

func assertRequestCountMetric(t *testing.T, gatherer prometheus.Gatherer) {
	t.Helper()
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, family := range metricFamilies {
		if family.GetName() != "http_server_requests_total" {
			continue
		}
		if len(family.Metric) != 1 {
			t.Fatalf("expected one request count series, got %d", len(family.Metric))
		}

		metric := family.Metric[0]
		if count := metric.GetCounter().GetValue(); count != 2 {
			t.Fatalf("expected request count 2, got %v", count)
		}

		labels := make(map[string]string, len(metric.Label))
		for _, label := range metric.Label {
			labels[label.GetName()] = label.GetValue()
		}
		for name, expected := range map[string]string{
			"method": http.MethodGet,
			"route":  "/things/{thingID}",
			"status": "204",
		} {
			if actual := labels[name]; actual != expected {
				t.Fatalf("expected label %s=%q, got %q", name, expected, actual)
			}
		}
		return
	}

	t.Error("request count metric was not registered")
}

func assertRequestDurationMetric(t *testing.T, gatherer prometheus.Gatherer) {
	t.Helper()
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, family := range metricFamilies {
		if family.GetName() != "http_server_request_duration_seconds" {
			continue
		}
		if len(family.Metric) != 1 {
			t.Fatalf("expected one duration series, got %d", len(family.Metric))
		}
		if count := family.Metric[0].GetHistogram().GetSampleCount(); count != 2 {
			t.Fatalf("expected duration sample count 2, got %d", count)
		}
		return
	}

	t.Error("duration metric was not registered")
}
