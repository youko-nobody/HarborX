package remote

type Summary struct {
	ConnectionModes []string `json:"connectionModes"`
	Capabilities    []string `json:"capabilities"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Summary() Summary {
	return Summary{
		ConnectionModes: []string{"websocket", "http", "pull"},
		Capabilities: []string{
			"server-registry",
			"agent-heartbeat",
			"streamed-install-tasks",
			"xray-manage",
			"nginx-manage",
			"warp-manage",
		},
	}
}

