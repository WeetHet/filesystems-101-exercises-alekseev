package client

import (
	"context"
	"net/http"
	"strings"

	"tests-runner/util/errors"
	"tests-runner/util/server/headers"
)

// Params represent authentication request
type Params struct {
	// Bearer or Admin token. Token without "Bearer " or "Admin "
	// prefix are treated as JWT tokens.
	Token string
	// OnBehalfToken token can be used instead of regular Token
	// so that request is authenticated like it was made by
	// user specified in OnBehalfExtID.
	//
	// Only Admin (with Admin prefix) tokens are supported for
	// on behalf auth.
	OnBehalfToken string
	OnBehalfExtID string
	// OnBehalfACL might carry serialized ACL of an OnBehalf user.
	OnBehalfACL string
	IP          string // current request user IP
	// ACLHash is extracted from the X-Acl-Hash header and forwarded to the
	// IAM proxy for cache validation.
	ACLHash string
}

// IsOnBehalfAuth tells if Params carry on behalf auth. token.
func (p *Params) IsOnBehalfAuth() bool {
	return p.Token == "" && p.OnBehalfToken != "" && p.OnBehalfExtID != ""
}

// Validate verifies authentication parameters
func (p *Params) Validate() error {
	if p.Token == "" && p.OnBehalfToken == "" {
		return errors.New("auth token is missing")
	}
	if p.Token != "" && (p.OnBehalfToken != "" || p.OnBehalfExtID != "" || p.OnBehalfACL != "") {
		return errors.New("cannot use both token and on behalf token")
	}
	if p.OnBehalfToken != "" && p.OnBehalfExtID == "" {
		return errors.New("on behalf user not specified")
	}
	if p.OnBehalfToken != "" && !strings.HasPrefix(p.OnBehalfToken, "Admin ") {
		return errors.New("only admin tokens are supported for on behalf auth")
	}
	return nil
}

// Authenticator is authenticate users making requests and put
// authenticated users info into request context
//
// See user subpackage for details on how to extract user info from
// modified context.
type Authenticator interface {
	Authenticate(ctx context.Context, p *Params) (context.Context, error)
}

// ExtractAuthParamsFromHeader parses auth parameters from request headers
func ExtractAuthParamsFromHeader(r *http.Request) (p *Params, err error) {
	p = &Params{
		Token:         r.Header.Get(headers.Authorization),
		OnBehalfToken: r.Header.Get(headers.AuthorizationOnBehalf),
		OnBehalfExtID: r.Header.Get(headers.AuthorizationOnBehalfExtID),
		OnBehalfACL:   r.Header.Get(headers.AuthorizationOnBehalfACL),
		ACLHash:       r.Header.Get(headers.ACLHash),
	}

	webSockToken := r.Header.Get("Sec-WebSocket-Protocol")
	if webSockToken != "" {
		if p.Token != "" {
			return nil, errors.New("cannot use both token and websock token")
		}
		p.Token = strings.TrimSpace(webSockToken)
	}

	return p, nil
}
