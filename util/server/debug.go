//go:build !srv_no_debug

package server

import (
	"context"
	"net/http/pprof"
	"os"

	"cloud.google.com/go/profiler"
	"github.com/google/gops/agent"
	"github.com/spf13/pflag"

	"tests-runner/util/errors"
	"tests-runner/util/util/errutil"
)

// DebugFlags contains command line flags needed for debug
type DebugFlags struct {
	EnableGProf        bool   `json:"enable_grpof" arg:"env:SERVER_ENABLE_GPROF"` // enable google cloud Stackdriver profiler
	GopsAddr           string `json:"gops_addr" arg:"env:SERVER_GOPS_ADDR"`
	EnablePProf        bool   `json:"enable_pprof" arg:"env:SERVER_ENABLE_PPROF"`
	LogRequests        bool   `json:"log_requests" arg:"env:SERVER_LOG_REQUESTS"`
	LogRequestsHeaders bool   `json:"log_requests_headers" arg:"env:SERVER_LOG_REQUESTS_HEADERS"`
	// memory usage monitor: heap profiles are uploaded to google storage
	// once process RSS goes over the threshold
	MemUsageBucket      string `json:"mem_usage_bucket" arg:"env:SERVER_MEM_USAGE_BUCKET"`
	MemUsageBucketCreds string `json:"mem_usage_bucket_creds" arg:"env:SERVER_MEM_USAGE_BUCKET_CREDS"`
	MemUsageThresholdMB int    `json:"mem_usage_threshold_mb" arg:"env:SERVER_MEM_USAGE_THRESHOLD_MB"`
}

// DebugConfig describes server config related to debug
type DebugConfig struct {
	// debug options
	GopsAddr           string
	EnableGProf        bool
	EnablePProf        bool
	LogRequests        bool
	LogRequestsHeaders bool
	// memory usage monitor options
	MemUsageBucket      string
	MemUsageBucketCreds string
	MemUsageThresholdMB int
}

func injectDebugFlags(fs *pflag.FlagSet, f *DebugFlags) {
	fs.StringVar(&f.GopsAddr, "server.gops.addr", "127.0.0.1:9990", "gops agent address")
	fs.BoolVar(&f.EnableGProf, "server.gprof", false, "Enable GCP Stackdriver profiler")
	fs.BoolVar(&f.EnablePProf, "server.pprof", false, "Enable pprof HTTP endpoints")
	fs.BoolVar(&f.LogRequests, "server.log.requests", false, "Enable requests logging")
	fs.BoolVar(&f.LogRequestsHeaders, "server.log.requests.headers", false, "Enable requests headers logging")
	fs.StringVar(&f.MemUsageBucket, "server.memusage.bucket", "",
		"Google storage bucket to upload heap profiles to when memory usage is too high (enables the memory usage monitor)")
	fs.StringVar(&f.MemUsageBucketCreds, "server.memusage.bucket.creds", "",
		"Path to the service account JSON key used to upload heap profiles")
	fs.IntVar(&f.MemUsageThresholdMB, "server.memusage.threshold.mb", 0,
		"Process RSS in MB above which heap profiles are taken")
}

func debugConfigFromFlags(f *DebugFlags) (cfg DebugConfig) {
	errutil.BugOn(f.LogRequestsHeaders && os.Getenv("TESTING") != "1",
		"--server.log.requests.headers is only allowed in testing deployments due to security reasons")

	return DebugConfig{
		GopsAddr:           f.GopsAddr,
		EnableGProf:        f.EnableGProf,
		EnablePProf:        f.EnablePProf,
		LogRequests:        f.LogRequests,
		LogRequestsHeaders: f.LogRequestsHeaders,

		MemUsageBucket:      f.MemUsageBucket,
		MemUsageBucketCreds: f.MemUsageBucketCreds,
		MemUsageThresholdMB: f.MemUsageThresholdMB,
	}
}

func (s *Server) startDebug(ctx context.Context) (err error) {
	if s.DisableMonitoring {
		return nil
	}

	if s.GopsAddr != "" {
		err := agent.Listen(agent.Options{Addr: s.GopsAddr})
		if err != nil {
			return errors.Wrap(err, "gops/agent.Listen")
		}
	}
	if s.EnableGProf {
		err := profiler.Start(profiler.Config{
			Service:        s.Name,
			ServiceVersion: s.Version,
			// alloc profiles take >1.5GB of RAM for some reason, so lead to OOMs
			NoAllocProfiling: true,
		})
		if err != nil {
			return errors.Wrap(err, "cannot start google profiler")
		}
	}
	if s.EnablePProf {
		s.Lock()
		s.rMgmt.HandleFunc("/debug/pprof/", pprof.Index)
		s.rMgmt.HandleFunc("/debug/pprof/mutex", pprof.Index)
		s.rMgmt.HandleFunc("/debug/pprof/heap", pprof.Index)
		s.rMgmt.HandleFunc("/debug/pprof/block", pprof.Index)
		s.rMgmt.HandleFunc("/debug/pprof/threadcreate", pprof.Index)
		s.rMgmt.HandleFunc("/debug/pprof/goroutine", pprof.Index)
		s.rMgmt.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.rMgmt.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.rMgmt.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.Unlock()
	}
	return nil
}
