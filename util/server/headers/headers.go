package headers

const (
	// Authorization header carries user token.
	Authorization = "Authorization"
	// AuthorizationOnBehalf header carries user token in case on-behalf
	// authorization is used (AuthorizationOnBehalfExtID must also be provided).
	AuthorizationOnBehalf = "X-Authorization-Onbehalf-Token"
	// AuthorizationOnBehalfExtID header carries external ID of a user
	// to pretend as.
	AuthorizationOnBehalfExtID = "X-Authorization-Onbehalf-Extid"
	// AuthorizationOnBehalfACL headers carries serialized ACL of a user.
	// This header is only allowed for on-behalf authorization.
	AuthorizationOnBehalfACL = "X-Authorization-Onbehalf-Acl"
	// XForwardedFor is standard HTTP Header to track client IP address
	// It is normally set by load balancers: https://datatracker.ietf.org/doc/html/rfc7239
	XForwardedFor = "X-Forwarded-For"
	// ACLHash carries a hash of the caller's last known ACL.
	ACLHash = "X-Acl-Hash"
)
