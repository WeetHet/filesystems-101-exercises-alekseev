package sctx

import (
	"context"

	"tests-runner/util/authn/user"
	"tests-runner/util/errors"
	"tests-runner/util/iam"
)

// userKey is a key used to store struct User in a context
type userKey struct{}

// User retrieves user.User from a context
func User(ctx context.Context) (*user.User, error) {
	v := ctx.Value(userKey{})
	user, _ := v.(*user.User)
	if user == nil {
		return nil, errors.ErrUnauthenticated.WithMessage("user not found in context")
	}
	return user, nil
}

// WithUser puts a user.User to a context so that it can be retrieved
// with User() later
func WithUser(ctx context.Context, user *user.User) context.Context {
	if user == nil {
		panic("sctx.WithUser() is not supposed to clear *User")
	}
	return context.WithValue(ctx, userKey{}, user)
}

// aclCtxKey is a key used to store an iam.ACL in a context
type aclCtxKey struct{}

// ACL retrieves iam.ACL from a context
func ACL(ctx context.Context) (*iam.ACL, error) {
	v := ctx.Value(aclCtxKey{})
	acl, _ := v.(*iam.ACL)
	if acl == nil {
		return nil, errors.ErrUnauthenticated.WithMessage("ACL not found in context")
	}
	return acl, nil
}

// WithACL puts specified acl into context so it can be retrieved
// with ACL() later
func WithACL(ctx context.Context, acl *iam.ACL) context.Context {
	if acl == nil {
		panic("sctx.WithACL() is not supposed to clear ACL")
	}
	return context.WithValue(ctx, aclCtxKey{}, acl)
}

type ipCtxKey struct{}

// WithActorIP puts specified ip into context so it can be retrieved
// with ActorIP() later
func WithActorIP(ctx context.Context, ip string) context.Context {
	if ip == "" {
		panic("sctx.WithActorIP() is not supposed to empty IP")
	}
	return context.WithValue(ctx, ipCtxKey{}, ip)
}

// ActorIP retrieves actor IP from a context
func ActorIP(ctx context.Context) string {
	v := ctx.Value(ipCtxKey{})
	ip, _ := v.(string)
	return ip
}
