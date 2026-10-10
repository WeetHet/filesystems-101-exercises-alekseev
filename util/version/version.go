// Package version contains version information for the app.
//
// The application version must be set by a build script that
// calls the go compiler with the following options:
//
//	-ldflags '-X tests-runner/util/version.Version=$(VERSION)'
//
// NOTE: `go build` does not concatenante repeated -ldflags
// parameters. It is a job of a build script to construct
// a value to the only -ldflags option.
package version

// Version is set by the build script.
var Version = "was not built properly"
