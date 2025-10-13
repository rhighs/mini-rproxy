package pluginapi

import (
	"net/http"
)

// Phase indicates whether the plugin is being invoked before the upstream
// request (PhaseRequest) or after the upstream response is received
// (PhaseResponse).
type Phase int

const (
	PhaseRequest Phase = iota
	PhaseResponse
)

// PhaseRequest:
//   - Request is non-nil
//   - Response is nil
//
// PhaseResponse:
//   - Request and Response are non-nil (Response is the upstream response
//     before being written to the client).
//
// Values is just a scratchpad map shared across phases.
type Context struct {
	Phase    Phase
	Request  *http.Request
	Response *http.Response
	Values   map[string]any
}

// Plugin defines the runtime plugin interface.
// A single Handle method is invoked twice per request:
//  1. PhaseRequest  (before dialing upstream)
//  2. PhaseResponse (after upstream responded, before body is sent to client)
//
// Return an error to abort processing. If returned during PhaseRequest the
// proxy will not contact upstream. If returned during PhaseResponse the
// proxy will emit a 502 (or configurable) to the client.
type Plugin interface {
	Name() string
	Init()
	Handle(ctx *Context) error
}
