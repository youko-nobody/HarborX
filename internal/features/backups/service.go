package backups

type Summary struct {
	Targets      []string `json:"targets"`
	Capabilities []string `json:"capabilities"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		Targets:      []string{"database", "templates", "rule-sets", "settings", "xray-snapshots"},
		Capabilities: []string{"export", "restore", "retention", "audit-log"},
	}
}

