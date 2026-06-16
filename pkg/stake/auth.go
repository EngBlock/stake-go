package stake

// CredentialsLoginRequest authenticates with username/password credentials.
type CredentialsLoginRequest struct {
	Username       string  `json:"username"`
	Password       string  `json:"password"`
	OTP            *string `json:"otp,omitempty"`
	RememberMeDays int     `json:"rememberMeDays"`
	PlatformType   string  `json:"platformType"`
}

func (r CredentialsLoginRequest) withDefaults() *CredentialsLoginRequest {
	if r.RememberMeDays == 0 {
		r.RememberMeDays = 30
	}
	if r.PlatformType == "" {
		r.PlatformType = defaultPlatformType
	}
	return &r
}

// SessionTokenLoginRequest authenticates with an existing Stake session token.
type SessionTokenLoginRequest struct {
	Token string `json:"token"`
}
