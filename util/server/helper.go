package server

import (
	"net/http"
	"strings"

	"tests-runner/util/server/headers"
)

// GetClientIP get ip address of client from http request
func GetClientIP(r *http.Request) (ip string) {
	ip = GetClientForwardIP(r)
	if ip == "" {
		ip = r.RemoteAddr
		// The HTTP server sets RemoteAddr to an "IP:port" address before invoking a handler.
		// See RemoteAddr docs.
		if portStart := strings.LastIndexByte(ip, ':'); portStart != -1 {
			ip = ip[:portStart]
		}
	}
	return ip
}

// GetClientForwardIP get only forward ip address from http request
func GetClientForwardIP(r *http.Request) (ip string) {
	if fwd := r.Header.Get(headers.XForwardedFor); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	return ip
}
