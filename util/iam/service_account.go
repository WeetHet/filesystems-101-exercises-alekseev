package iam

import (
	"strings"

	"tests-runner/util/ulid"
)

// Predefined admin service account.
//
// It is "virtual" because it is not stored anywhere,
// but injected automatically into request context in case
// request arrives with super admin token.
const (
	AccountAdminName  = "admin"
	AccountAdminEmail = "admin@service.account" // == extID also
)

// AccountAdminID is a artificial identifier of a super admin account
var AccountAdminID = ulid.MustParse("00000000000000000000000666")

// IsServiceAccount returns true if account email belongs to service account
func IsServiceAccount(accountEmail string) bool {
	return strings.HasSuffix(accountEmail, ".service.account")
}
