package dashboard

import "harborx/internal/features/catalog"

type Summary struct {
	ModulesTotal      int      `json:"modulesTotal"`
	ModulesInProgress int      `json:"modulesInProgress"`
	FocusAreas        []string `json:"focusAreas"`
	PlatformMode      string   `json:"platformMode"`
	GatingModel       string   `json:"gatingModel"`
}

type Service struct {
	catalog catalog.Service
}

func NewService(catalogService catalog.Service) Service {
	return Service{catalog: catalogService}
}

func (s Service) Summary() Summary {
	modules := s.catalog.Modules()
	inProgress := 0
	for _, module := range modules {
		if module.Status == "in_progress" {
			inProgress++
		}
	}

	return Summary{
		ModulesTotal:      len(modules),
		ModulesInProgress: inProgress,
		FocusAreas: []string{
			"Rules Studio",
			"Templates",
			"Nodes and subscriptions",
			"Remote management",
		},
		PlatformMode: "selfhost",
		GatingModel:  "none",
	}
}
