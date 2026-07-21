package traffic

import (
	"errors"
	"sort"
	"strings"
)

type Summary struct {
	Metrics      []string `json:"metrics"`
	Capabilities []string `json:"capabilities"`
}

type Sample struct {
	ID          string         `json:"id"`
	SampleScope string         `json:"sampleScope"`
	ScopeID     string         `json:"scopeId"`
	RXBytes     int64          `json:"rxBytes"`
	TXBytes     int64          `json:"txBytes"`
	Rate        map[string]any `json:"rate"`
	RecordedAt  string         `json:"recordedAt"`
}

type Rollup struct {
	SampleScope string `json:"sampleScope"`
	ScopeID     string `json:"scopeId"`
	RXBytes     int64  `json:"rxBytes"`
	TXBytes     int64  `json:"txBytes"`
	Samples     int    `json:"samples"`
	LastSeenAt  string `json:"lastSeenAt"`
}

type CreateSampleInput struct {
	SampleScope string         `json:"sampleScope"`
	ScopeID     string         `json:"scopeId"`
	RXBytes     int64          `json:"rxBytes"`
	TXBytes     int64          `json:"txBytes"`
	Rate        map[string]any `json:"rate"`
}

type Repository interface {
	ListTrafficSamples(scope string, scopeID string) ([]Sample, error)
	CreateTrafficSample(input CreateSampleInput) (Sample, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Metrics:      []string{"user-usage", "node-usage", "server-usage", "rate", "history"},
		Capabilities: []string{"dashboard-summary", "30-day-history", "aggregation"},
	}
}

func (s Service) ListSamples(scope string, scopeID string) ([]Sample, error) {
	if s.repo == nil {
		return nil, errors.New("traffic repository is not configured")
	}
	return s.repo.ListTrafficSamples(scope, scopeID)
}

func (s Service) CreateSample(input CreateSampleInput) (Sample, error) {
	if s.repo == nil {
		return Sample{}, errors.New("traffic repository is not configured")
	}
	if strings.TrimSpace(input.SampleScope) == "" {
		return Sample{}, errors.New("traffic sample scope is required")
	}
	if strings.TrimSpace(input.ScopeID) == "" {
		return Sample{}, errors.New("traffic scope id is required")
	}
	if input.RXBytes < 0 || input.TXBytes < 0 {
		return Sample{}, errors.New("traffic bytes cannot be negative")
	}
	return s.repo.CreateTrafficSample(input)
}

func (s Service) Rollups(scope string, scopeID string) ([]Rollup, error) {
	if s.repo == nil {
		return nil, errors.New("traffic repository is not configured")
	}
	samples, err := s.repo.ListTrafficSamples(scope, scopeID)
	if err != nil {
		return nil, err
	}
	rollupsByKey := map[string]*Rollup{}
	for _, sample := range samples {
		key := sample.SampleScope + "\x00" + sample.ScopeID
		item := rollupsByKey[key]
		if item == nil {
			item = &Rollup{SampleScope: sample.SampleScope, ScopeID: sample.ScopeID}
			rollupsByKey[key] = item
		}
		item.RXBytes += sample.RXBytes
		item.TXBytes += sample.TXBytes
		item.Samples++
		if sample.RecordedAt > item.LastSeenAt {
			item.LastSeenAt = sample.RecordedAt
		}
	}
	items := make([]Rollup, 0, len(rollupsByKey))
	for _, item := range rollupsByKey {
		items = append(items, *item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].RXBytes+items[i].TXBytes > items[j].RXBytes+items[j].TXBytes
	})
	return items, nil
}
