package notifications

type Summary struct {
	Channels     []string `json:"channels"`
	Capabilities []string `json:"capabilities"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		Channels:     []string{"telegram"},
		Capabilities: []string{"alerts", "daily-summary", "traffic-thresholds", "server-status"},
	}
}

