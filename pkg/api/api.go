// ----------------------------------------------------------------------------
// Copyright (c) Ben Coleman, 2020
// Licensed under the MIT License.
//
// Base API that all services implement and extend
// ----------------------------------------------------------------------------

package api

// Base holds a standard set of values for all services & APIs
type Base struct {
	ServiceName string
	Healthy     bool
	Version     string
	BuildInfo   string
}

// NewBase creates and returns a new Base API instance
func NewBase(name, ver, info string, healthy bool) *Base {
	return &Base{
		ServiceName: name,
		Healthy:     healthy,
		Version:     ver,
		BuildInfo:   info,
	}
}
