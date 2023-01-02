package logging

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/go-chi/chi/middleware"
)

// NewFilteredRequestLogger constructs a new FilteredRequestLogger with stdout logging
func NewFilteredRequestLogger(filterOut *regexp.Regexp) func(next http.Handler) http.Handler {
	formatter := middleware.DefaultLogFormatter{
		Logger:  log.New(os.Stdout, "", log.LstdFlags),
		NoColor: false,
	}

	return FilteredRequestLogger(&formatter, filterOut)
}

// FilteredRequestLogger is a copy of the middleware.RequestLogger function
// - But with a filter to exclude certain URLs from logging
func FilteredRequestLogger(f middleware.LogFormatter, filterOut *regexp.Regexp) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Extra logic to filter out certain URLs from logging
			if filterOut.MatchString(r.URL.String()) {
				next.ServeHTTP(w, r)
				return
			}

			entry := f.NewLogEntry(r)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				entry.Write(ww.Status(), ww.BytesWritten(), ww.Header(), time.Since(t1), nil)
			}()

			next.ServeHTTP(ww, middleware.WithLogEntry(r, entry))
		}

		return http.HandlerFunc(fn)
	}
}
