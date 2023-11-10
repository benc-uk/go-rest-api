// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2023
// Licensed under the MIT License.
//
// Server Side Events (SSE) package for Go
// ----------------------------------------------------------------------------

package sse

import (
	"fmt"
	"io"
)

// Dead simple struct to support SSE format
type SSE struct {
	Event string
	Data  string
	ID    string
}

// Write the SSE format message to a writer
func (sse *SSE) Write(w io.Writer) {
	if sse.Event != "" {
		fmt.Fprintf(w, "event: %s\n", sse.Event)
	}

	if sse.ID != "" {
		fmt.Fprintf(w, "id: %s\n", sse.ID)
	}

	fmt.Fprintf(w, "data: %s\n\n", sse.Data)
}
