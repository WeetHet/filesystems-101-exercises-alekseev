package client

import (
	"os"

	"github.com/spf13/pflag"
)

// OIDCFlags contains flags needed to setup OIDC token verification
type OIDCFlags struct {
	OidcProvider string `json:"oidc_provider" arg:"env:OIDC_PROVIDER"`
	OidcClientID string `json:"oidc_client_id" arg:"env:OIDC_CLIENT_ID"`
}

// AdminFlags contain flags needed to setup inter-services super admin authentication
type AdminFlags struct {
	// AdminKey can be a comma separated list of keys
	AdminKey string `json:"auth_admin_key" arg:"env:AUTH_ADMIN_KEY"`
}

// Flags contains command line flags needed to instantiate OIDC client
type Flags struct {
	OIDCFlags
	AdminFlags
}

// InjectFlags injects OIDC related flags into the given flag set.
func InjectFlags(fs *pflag.FlagSet, f *Flags) {
	fs.StringVar(&f.OidcProvider, "oidc.provider", "", "OIDC provider")
	fs.StringVar(&f.OidcClientID, "oidc.client-id", "", "OIDC client ID")

	fs.StringVar(&f.AdminKey, "auth.admin.key", os.Getenv("AUTH_ADMIN_KEY"), "Auth admin API keys (separated by comma)")
}
