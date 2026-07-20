package certificates

type Summary struct {
	Providers     []string `json:"providers"`
	Capabilities  []string `json:"capabilities"`
	DeploymentMode string  `json:"deploymentMode"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		Providers: []string{"cloudflare", "alidns", "dnspod", "tencent", "godaddy", "namesilo", "webroot"},
		Capabilities: []string{"issue", "renew", "deploy", "upload", "auto-renew"},
		DeploymentMode: "master-and-remote",
	}
}

