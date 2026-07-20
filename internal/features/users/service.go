package users

type Summary struct {
	Roles          []string `json:"roles"`
	SupportsLimits bool     `json:"supportsLimits"`
	SupportsPrefs  bool     `json:"supportsPrefs"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		Roles:          []string{"admin", "member"},
		SupportsLimits: true,
		SupportsPrefs:  true,
	}
}

