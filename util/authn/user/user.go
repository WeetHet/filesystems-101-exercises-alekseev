package user

// microsoft and google provider is set up in dex config
const (
	// ProviderInternal is used for Federated.Provider when, for example, on-behalf authentication is used.
	ProviderInternal = "internal"
	// ProviderMicrosoft is used for Federated.Provider when, for example, microsoft oauth used.
	ProviderMicrosoft = "microsoft"
	// ProviderGoogle is used for Federated.Provider when, for example, google oauth used.
	ProviderGoogle = "google"
)

// Federated contains claims federated from upstream authentication provider.
type Federated struct {
	Provider string `json:"provider"`
	UserID   string `json:"user_id"`
}

// User describes an authenticated user
type User struct {
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	EmailVerified    bool      `json:"email_verified"`
	Federated        Federated `json:"federated"`
	ImpersonatedFrom string    `json:"impersonated_from"`
}
