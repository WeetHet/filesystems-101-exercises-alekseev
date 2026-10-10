package iam

//go:generate enumer -type Role -trimprefix Role -transform snake

// Role is a type for role which can be given to actor upon some
// access controllable object; It should be treated as an index
// of a bit in a roles bit string.
type Role uint16

// Predefined IAM roles
const (
	RoleAdmin Role = 0 // super admin used for service accounts (can do anything)
)

// AllRoles lists all available roles
var AllRoles = []Role{RoleAdmin}
