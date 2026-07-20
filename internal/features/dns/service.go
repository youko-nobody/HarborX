package dns

type Summary struct {
	Providers     []string `json:"providers"`
	Capabilities  []string `json:"capabilities"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		Providers:    []string{"cloudflare", "alidns", "dnspod", "tencent", "godaddy", "namesilo"},
		Capabilities: []string{"record-manage", "zone-lookup", "acme-support", "dynamic-dns"},
	}
}

