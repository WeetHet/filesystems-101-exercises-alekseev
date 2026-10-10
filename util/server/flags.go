package server

import (
	"github.com/spf13/pflag"
)

// ConfigFlags contains command line flags needed to start compound server
type ConfigFlags struct {
	Addr        string `arg:"env:SERVER_ADDR"`
	AddrMgmt    string `arg:"env:SERVER_ADDR_MGMT"`
	DisableCORS bool   `arg:"env:SERVER_DISABLE_CORS"`
}

// Flags contains command line flags needed to instantiate compound server
type Flags struct {
	ConfigFlags
	AuthFlags
	DebugFlags
}

// InjectFlags injects all server related flags into the given flag set.
func InjectFlags(fs *pflag.FlagSet, f *Flags) {
	fs.StringVarP(&f.Addr, "server.addr", "a", "localhost:9000", "HTTP listen address")
	fs.StringVar(&f.AddrMgmt, "server.addr.mgmt", "", "HTTP listen address for mgmt endpoints")
	fs.BoolVar(&f.DisableCORS, "server.cors.disable", false, "Disable CORS for all domains")
	injectAuthFlags(fs, &f.AuthFlags)
	injectDebugFlags(fs, &f.DebugFlags)
}

// ConfigFromFlags returns new server config based on flag settings
func ConfigFromFlags(f *Flags) (cfg Config) {
	return Config{
		BindAddr:     f.Addr,
		BindAddrMgmt: f.AddrMgmt,
		DisableCORS:  f.DisableCORS,
		AuthConfig:   authConfigFromFlags(&f.AuthFlags),
		DebugConfig:  debugConfigFromFlags(&f.DebugFlags),
	}
}
