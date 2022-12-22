// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020
// Licensed under the MIT License.
//
// RFC-7807 implementation for sending standard format API errors
// ----------------------------------------------------------------------------

package problem

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
)

// Problem in RFC-7807 format
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// New creates a RFC 7807 problem object
func New(typeStr string, title string, status int, detail, instance string) *Problem {
	return &Problem{typeStr, title, status, detail, instance}
}

// HTTPSend sends a RFC 7807 problem object as HTTP response
func (p *Problem) Send(resp http.ResponseWriter) {
	log.Printf("### 💥 API %s", p.Error())
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(p.Status)
	_ = json.NewEncoder(resp).Encode(p)
}

// Wrap creates a Problem wrapping an error
func Wrap(status int, typeStr string, instance string, err error) *Problem {
	var p *Problem
	if err != nil {
		p = New(typeStr, MyCaller(), status, err.Error(), instance)
	} else {
		p = New(typeStr, MyCaller(), status, "Other error occurred", instance)
	}

	return p
}

// Implement error interface
func (p Problem) Error() string {
	return fmt.Sprintf("Problem: Type: '%s', Title: '%s', Status: '%d', Detail: '%s', Instance: '%s'",
		p.Type, p.Title, p.Status, p.Detail, p.Instance)
}

// getFrame returns the stack frame at the given depth
func getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}

	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])

		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()

			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

// MyCaller returns the caller of the function that called it
func MyCaller() string {
	// Skip GetCallerFunctionName and the function to get the caller of
	return getFrame(2).Function
}
