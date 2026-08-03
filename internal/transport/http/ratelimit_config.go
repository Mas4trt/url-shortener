//go:build !integration

package httptransport

const (
	writeEndpointRate  = 5.0
	writeEndpointBurst = 20.0

	authEndpointRate  = 1.0
	authEndpointBurst = 5.0
)
