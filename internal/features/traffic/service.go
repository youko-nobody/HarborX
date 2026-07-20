package traffic

type Summary struct {
	Metrics      []string `json:"metrics"`
	Capabilities []string `json:"capabilities"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		Metrics:      []string{"user-usage", "node-usage", "server-usage", "rate", "history"},
		Capabilities: []string{"dashboard-summary", "30-day-history", "aggregation"},
	}
}

