package server

import (
	"context"
)

// SubServer is a common set for methods for all subservers (components)
type SubServer interface {
	// Start should run all needed background tasks asynchronously
	// Specified context `ctx` can be used as a context for background
	// operations.
	Start(ctx context.Context) (err error)
	// Stop should emit stop signals for all running background tasks
	Stop()
	// Wait should block until all background tasks are stopped
	// Will be called only after Stop() was called
	Wait()
	IsHealthy(ctx context.Context) (err error)
	IsReady(ctx context.Context) bool
}

// HTTPSubServer is a common interface, that all HTTP servers should implement
type HTTPSubServer interface {
	SubServer
	// AddRoutes should use `s.AddRoutes()` or `s.Router().XXX` to register
	// HTTP routes.
	AddRoutes(s *Server) (err error)
}
