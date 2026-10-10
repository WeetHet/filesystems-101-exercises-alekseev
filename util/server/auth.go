package server

import (
	"context"

	"github.com/spf13/pflag"

	authclient "tests-runner/util/authn"
	"tests-runner/util/authn/user"
	"tests-runner/util/errors"
	sctx "tests-runner/util/server/context"

	"tests-runner/util/iam"
)

// AuthFlags contains command line flags needed for authentication
type AuthFlags struct {
	authclient.Flags
	ACLServiceEP string `json:"acl_service" arg:"env:ACL_SERVICE"`
	DisableAuth  bool   `json:"disable_auth" arg:"env:SERVER_DISABLE_AUTH"`
}

// AuthConfig contains all server config params
type AuthConfig struct {
	DisableAuth bool

	OidcProvider string
	OidcClientID string
	// AuthClient may be provided for server to authenticate requests
	// In case AuthC is not passed, when it is initialized from
	// OidcProvider/OidcClientID (in case DisableAuth is false)
	AuthClient authclient.Authenticator

	AdminKey string

	// ACLServiceEP is used to initialize ACL client which
	// uses IAM's REST API in case ACLGetter is not specified
	// explicitly.
	ACLServiceEP string
	// ACLGetter may be passed in a config or either would
	// be initialized from ACLServiceEP.
	//
	// Is disabled auth case, ACLGetter will be initialized
	// to return ACL only for iam admin account
	ACLGetter iam.ACLGetter
}

func injectAuthFlags(fs *pflag.FlagSet, f *AuthFlags) {
	fs.StringVar(&f.ACLServiceEP, "server.acl.service", "", "ACL service address")
	fs.BoolVar(&f.DisableAuth, "server.auth.disable", false, "Disable API authentication")
	authclient.InjectFlags(fs, &f.Flags)
}

func authConfigFromFlags(f *AuthFlags) (cfg AuthConfig) {
	return AuthConfig{
		DisableAuth: f.DisableAuth,

		OidcProvider: f.OidcProvider,
		OidcClientID: f.OidcClientID,
		ACLServiceEP: f.ACLServiceEP,

		AdminKey: f.AdminKey,
	}
}

func (s *Server) initAuth(ctx context.Context) (err error) {
	if s.DisableAuth {
		s.ACLGetter = NoAuthACLGetter
		s.AuthClient = NoAuthAuthenticator
		return nil
	}

	panic("removed")
}

// noAuthACLGetter implements iam.ACLGetter and used as
// a default ACL getter for server with disabled auth
type noAuthACLGetter struct {
}

var _ iam.ACLGetter = (*noAuthACLGetter)(nil)

func (c *noAuthACLGetter) Get(ctx context.Context, p iam.ACLGetParams) (acl *iam.ACL, err error) {
	if p.ActorExtID == iam.AccountAdminEmail || p.ActorID == iam.AccountAdminID {
		return iam.GetAdminACL(), nil
	}
	return nil, errors.ErrForbidden.WithMessagef("no such actor: ext_id = %q, id = %q", p.ActorExtID, p.ActorID)
}

// noIAMSvcACLGetter implements iam.ACLGetter for the case when no IAM service
// is configured. It handles only actors that authenticate with an admin key.
type noIAMSvcACLGetter = noAuthACLGetter

// noAuthAuthenticator implements authclient.Authenticator
// but always injects Admin user into the context.
type noAuthAuthenticator struct {
}

var _ authclient.Authenticator = (*noAuthAuthenticator)(nil)

func (c *noAuthAuthenticator) Authenticate(ctx context.Context, p *authclient.Params) (context.Context, error) {
	return sctx.WithUser(ctx, &user.User{
		Email: iam.AccountAdminEmail,
		Name:  iam.AccountAdminName,
		Federated: user.Federated{
			Provider: user.ProviderInternal,
			UserID:   iam.AccountAdminEmail,
		},
	}), nil
}

// for use in tests
var (
	NoAuthACLGetter     = &noAuthACLGetter{}
	NoAuthAuthenticator = &noAuthAuthenticator{}
)
