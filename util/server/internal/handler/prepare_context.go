package handler

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	authn "tests-runner/util/authn"
	"tests-runner/util/authn/user"
	"tests-runner/util/errors"
	"tests-runner/util/gracefully"
	"tests-runner/util/iam"
	"tests-runner/util/logger"
	sctx "tests-runner/util/server/context"
	"tests-runner/util/util/errutil"
)

// ContextBuilder is a middleware to be used to add various
// useful values to HTTP request contexts.
type ContextBuilder struct {
	AuthClient authn.Authenticator
	ACLGetter  iam.ACLGetter
}

// BuildContextParams parametrizes context preparation of for
// a specified handler.
type BuildContextParams struct {
	HTTPRequest   *http.Request
	ActorIP       string
	Auth          bool
	InjectACL     bool
	NoDefaultTags bool
}

func (p *BuildContextParams) validate() error {
	if p.HTTPRequest == nil {
		return errors.New("HTTPRequest must be specified")
	}
	if p.InjectACL && !p.Auth {
		return errors.New("cannot inject acl without authentication")
	}
	return nil
}

// Build derives a context from a specified one adding various needed values to it:
// ACL, User, logging fields and reporting tags. In case error occurs in the middle
// of context preparation, already prepared to the point context is returned together
// with an error.
func (p *ContextBuilder) Build(ctx context.Context, params BuildContextParams) (newCtx context.Context, err error) {
	if err = params.validate(); err != nil {
		return ctx, err
	}

	newCtx = ctx
	if !params.NoDefaultTags {
		newCtx = p.addDefaultTags(ctx, params)
	}

	if params.Auth {
		err = gracefully.Do(newCtx, func(ctx context.Context) error {
			var authParams *authn.Params
			newCtx, authParams, err = p.authenticate(ctx, params)
			if err != nil {
				return err
			}

			if params.InjectACL {
				newCtx, err = p.injectACL(newCtx, authParams)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return newCtx, err
		}
	}

	// we want IP injected not only for authenticated requests
	newCtx = p.injectActorIP(newCtx, params)

	return newCtx, nil
}

func (p *ContextBuilder) addDefaultTags(ctx context.Context, params BuildContextParams) context.Context {
	loggerFields := logger.Fields{}
	reportingTags := map[string]string{}

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		reportingTags["trace_id"] = sc.TraceID().String()
	}

	if params.ActorIP != "" {
		loggerFields["actor_ip"] = params.ActorIP
	}

	newCtx := logger.NewContext(ctx, loggerFields)
	return newCtx
}

func (p *ContextBuilder) authenticate(
	ctx context.Context, params BuildContextParams,
) (newCtx context.Context, authParams *authn.Params, err error) {
	if params.HTTPRequest != nil {
		authParams, err = authn.ExtractAuthParamsFromHeader(params.HTTPRequest)
		if err != nil {
			return ctx, nil, err
		}
	}
	authParams.IP = params.ActorIP

	newCtx, err = p.AuthClient.Authenticate(ctx, authParams)
	if err != nil {
		return ctx, authParams, err
	}

	_, err = sctx.User(newCtx)
	errutil.BugOn(err != nil, "no user in the context after Authenticate")

	return newCtx, authParams, nil
}

func (p *ContextBuilder) injectACL(ctx context.Context, authParams *authn.Params) (newCtx context.Context, err error) {
	panic("removed")
}

func (p *ContextBuilder) injectActorIP(ctx context.Context, params BuildContextParams) context.Context {
	if params.ActorIP == "" {
		return ctx
	}

	// it is not useful to expose an IP address of
	// an actor, which uses super admin token for auth.
	u, _ := sctx.User(ctx)
	if u != nil && isSuperAdminUser(u) {
		return ctx
	}

	// inject IP address for non-admin or non-authenticated actors.
	return sctx.WithActorIP(ctx, params.ActorIP)
}

func isSuperAdminUser(u *user.User) bool {
	return u.Email == iam.AccountAdminEmail && u.Federated.UserID == iam.AccountAdminEmail
}

func isServiceAccount(u *user.User) bool {
	return iam.IsServiceAccount(u.Email) && iam.IsServiceAccount(u.Federated.UserID)
}
