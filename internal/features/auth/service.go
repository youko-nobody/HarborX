package auth

type Summary struct {
	LoginModes   []string `json:"loginModes"`
	SessionStore string   `json:"sessionStore"`
	SupportsTOTP bool     `json:"supportsTotp"`
	SupportsAPITokens bool `json:"supportsApiTokens"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		LoginModes:        []string{"password", "api-token"},
		SessionStore:      "sqlite",
		SupportsTOTP:      true,
		SupportsAPITokens: true,
	}
}

