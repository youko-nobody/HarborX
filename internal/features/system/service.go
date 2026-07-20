package system

type Summary struct {
	Capabilities []string `json:"capabilities"`
	DefaultTheme string   `json:"defaultTheme"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		Capabilities: []string{
			"theme-settings",
			"dashboard-refresh",
			"default-template",
			"short-link-behavior",
			"operator-permissions",
		},
		DefaultTheme: "sand",
	}
}

