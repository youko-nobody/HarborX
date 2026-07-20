package proxygroups

type Summary struct {
	GroupKinds   []string `json:"groupKinds"`
	Capabilities []string `json:"capabilities"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		GroupKinds:   []string{"select", "url-test", "fallback", "load-balance", "relay"},
		Capabilities: []string{"crud", "ordering", "policy-binding"},
	}
}

