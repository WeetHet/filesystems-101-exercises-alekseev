package iam

import (
	"context"

	"tests-runner/util/errors"
	"tests-runner/util/ulid"
)

// ACLGetter is responsible for retrieving actor ACLs.
type ACLGetter interface {
	Get(ctx context.Context, params ACLGetParams) (acl *ACL, err error)
}

// ACLGetParams describes actor to get ACL for.
type ACLGetParams struct {
	ActorID    ulid.ULID
	ActorExtID string
	// One of IgnoreActorIP or ActorIP must be specified.
	// It is done so to avoid missing UserIP parameter by accident.
	IgnoreActorIP bool
	ActorIP       string
	// Hash is the caller's last known ACL hash, forwarded to the IAM
	// proxy for cache validation. See GetACLParams.Hash.
	Hash string
}

// Validate checks parameters to be valid.
func (p *ACLGetParams) Validate() error {
	if p.ActorID == ulid.Nil && p.ActorExtID == "" {
		return errors.New("at least one of ActorID and ActorExtID must be specified")
	}
	if (p.ActorIP == "" && !p.IgnoreActorIP) || (p.ActorIP != "" && p.IgnoreActorIP) {
		return errors.New("one of ActorIP and IgnoreActorIP must be specified")
	}
	return nil
}
