package xray

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"harborx/internal/features/nodes"
	"harborx/internal/features/rules"
)

type Summary struct {
	Capabilities []string `json:"capabilities"`
	SnapshotMode string   `json:"snapshotMode"`
}

type Repository interface {
	ListNodes() ([]nodes.Node, error)
	ListRuleSets() ([]rules.RuleSet, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Capabilities: []string{"render-config", "preview-diff", "snapshot-history", "apply-plan"},
		SnapshotMode: "sqlite-and-filesystem",
	}
}

type Preview struct {
	Content string `json:"content"`
	Summary string `json:"summary"`
}

type xrayConfig struct {
	Log       map[string]string `json:"log"`
	Inbounds  []xrayInbound     `json:"inbounds"`
	Outbounds []xrayOutbound    `json:"outbounds"`
	Routing   xrayRouting       `json:"routing"`
}

type xrayInbound struct {
	Tag      string         `json:"tag"`
	Listen   string         `json:"listen"`
	Port     int            `json:"port"`
	Protocol string         `json:"protocol"`
	Settings map[string]any `json:"settings"`
}

type xrayOutbound struct {
	Tag      string         `json:"tag"`
	Protocol string         `json:"protocol"`
	Settings map[string]any `json:"settings"`
}

type xrayRouting struct {
	DomainStrategy string      `json:"domainStrategy"`
	Rules          []xrayRoute `json:"rules"`
}

type xrayRoute struct {
	Type        string   `json:"type"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
	OutboundTag string   `json:"outboundTag"`
}

func (s Service) Preview() (Preview, error) {
	if s.repo == nil {
		return Preview{}, errors.New("xray repository is not configured")
	}
	nodeItems, err := s.repo.ListNodes()
	if err != nil {
		return Preview{}, err
	}
	ruleSets, err := s.repo.ListRuleSets()
	if err != nil {
		return Preview{}, err
	}

	config := xrayConfig{
		Log: map[string]string{"loglevel": "warning"},
		Inbounds: []xrayInbound{
			{
				Tag:      "socks-in",
				Listen:   "127.0.0.1",
				Port:     10808,
				Protocol: "socks",
				Settings: map[string]any{"udp": true},
			},
			{
				Tag:      "http-in",
				Listen:   "127.0.0.1",
				Port:     10809,
				Protocol: "http",
				Settings: map[string]any{},
			},
		},
		Outbounds: buildOutbounds(nodeItems),
		Routing: xrayRouting{
			DomainStrategy: "IPIfNonMatch",
			Rules:          buildRoutes(ruleSets),
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return Preview{}, err
	}

	return Preview{
		Content: string(data),
		Summary: fmt.Sprintf(
			"%d inbounds, %d outbounds, %d routing rules",
			len(config.Inbounds),
			len(config.Outbounds),
			len(config.Routing.Rules),
		),
	}, nil
}

func buildOutbounds(items []nodes.Node) []xrayOutbound {
	outbounds := []xrayOutbound{
		{Tag: "direct", Protocol: "freedom", Settings: map[string]any{}},
		{Tag: "block", Protocol: "blackhole", Settings: map[string]any{}},
	}
	sorted := append([]nodes.Node(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, item := range sorted {
		if !item.Enabled {
			continue
		}
		outbounds = append(outbounds, xrayOutbound{
			Tag:      item.Name,
			Protocol: item.Protocol,
			Settings: map[string]any{
				"address": item.ServerHost,
				"port":    item.ServerPort,
			},
		})
	}
	return outbounds
}

func buildRoutes(ruleSets []rules.RuleSet) []xrayRoute {
	var routes []xrayRoute
	for _, set := range ruleSets {
		for _, rule := range set.Rules {
			if !rule.Enabled || rule.RuleType == "MATCH" {
				continue
			}
			route := xrayRoute{Type: "field", OutboundTag: normalizeOutbound(rule.Policy)}
			switch rule.RuleType {
			case "DOMAIN":
				route.Domain = []string{"full:" + rule.Pattern}
			case "DOMAIN-SUFFIX":
				route.Domain = []string{"domain:" + rule.Pattern}
			case "DOMAIN-KEYWORD":
				route.Domain = []string{"keyword:" + rule.Pattern}
			case "GEOIP":
				route.IP = []string{"geoip:" + rule.Pattern}
			case "IP-CIDR", "SRC-IP-CIDR":
				route.IP = []string{rule.Pattern}
			default:
				continue
			}
			routes = append(routes, route)
		}
	}
	return routes
}

func normalizeOutbound(policy string) string {
	switch policy {
	case "DIRECT", "Domestic":
		return "direct"
	case "REJECT":
		return "block"
	default:
		return "Proxy"
	}
}
