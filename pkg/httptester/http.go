// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020-2022
// Licensed under the MIT License.
//
// Helper for running API/route based tests
// ----------------------------------------------------------------------------

package httptester

import (
	"io"
	"log"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type TestCase struct {
	Name           string // Name of this test
	URL            string // URL to test
	Method         string // HTTP method to use
	Body           string // Body to send (if POST etc)
	CheckBody      string // Regex to check for in response body
	CheckBodyCount int    // Number of times regex should match
	CheckStatus    int    // Expected HTTP status code
}

func Run(t *testing.T, router chi.Router, testCases []TestCase) {
	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {

			log.Printf("### Running test: %s %s", test.Method, test.URL)
			req := httptest.NewRequest(test.Method, test.URL, strings.NewReader(test.Body))

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Content-Length", strconv.Itoa(len(test.Body)))

			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != test.CheckStatus {
				t.Errorf("Got status %d wanted %d", rec.Result().StatusCode, test.CheckStatus)
				return
			}

			if test.CheckBody != "" {
				body, _ := io.ReadAll(rec.Result().Body)
				bodyCheckRegex := regexp.MustCompile(test.CheckBody)
				matches := bodyCheckRegex.FindAllStringIndex(string(body), -1)

				if len(matches) != test.CheckBodyCount {
					t.Errorf("'%s' not found %d times in body ", test.CheckBody, test.CheckBodyCount)
					t.Logf(" BODY: %s", body)
					return
				}
			}
		})
	}
}
