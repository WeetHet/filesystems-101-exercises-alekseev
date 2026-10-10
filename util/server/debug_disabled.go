//go:build srv_no_debug

package server

import (
	"context"

	"github.com/spf13/pflag"
)

// DebugFlags contains command line flags needed for debug
type DebugFlags struct{}

// DebugConfig describes server config related to debug
type DebugConfig struct {
	LogRequests        bool
	LogRequestsHeaders bool
}

func injectDebugFlags(fs *pflag.FlagSet, f *DebugFlags)    {}
func debugConfigFromFlags(f *DebugFlags) (cfg DebugConfig) { return }

func (s *Server) startDebug(_ context.Context) error { return nil }
