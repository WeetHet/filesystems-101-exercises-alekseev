//go:build !srv_no_auth

package iam

import (
	"net/http"

	"tests-runner/util/errors"
	"tests-runner/util/ulid"
)

// ErrForbidden is used to designate that requested access is not allowed
var ErrForbidden = errors.Error{
	HTTPStatus: http.StatusForbidden,
	Code:       errors.CodeForbidden,
	Message:    "Access denied, insufficient permissions to access the object",
}.WithFatalTaskDetails()

// GetAdminACL returns super admin ACL allowing access to anything
func GetAdminACL() (acl *ACL) {
	return &ACL{
		ActorID:   AccountAdminID,
		Actor:     AccountAdminEmail,
		ActorName: AccountAdminName,
	}
}

type AccessControl struct {
}

// ACL controls user (actor) access scopes
type ACL struct {
	// Actor is the user account email
	Actor string `json:"actor,omitempty"`
	// ActorID is an identifier of a user, which has email written in Actor field
	ActorID ulid.ULID `json:"actor_id,omitempty"`
	// ActorName is a human-readable name of that user
	ActorName string `json:"actor_name,omitempty"`
	// AccessControls are access rules for objects with
	// specific IDs and types
	AccessControls []AccessControl `json:"access_controls,omitempty"`
}

// NewACL return new empty ACL
func NewACL() *ACL {
	return &ACL{}
}

// CheckAdmin checks that actor of the acl is admin (our internal service accout)
func (acl *ACL) CheckAdmin() (err error) {
	if acl.ActorID == AccountAdminID {
		return nil
	}
	return errors.Wrap(ErrForbidden, "acl.CheckAdmin()") // wrap to catch the call trace
}
