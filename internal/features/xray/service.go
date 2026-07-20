package xray

type Summary struct {
	Capabilities []string `json:"capabilities"`
	SnapshotMode string   `json:"snapshotMode"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		Capabilities: []string{"render-config", "preview-diff", "snapshot-history", "apply-plan"},
		SnapshotMode: "sqlite-and-filesystem",
	}
}

