package xray

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"harborx/internal/features/nodes"
	"harborx/internal/features/ops"
	"harborx/internal/features/rules"
)

type Summary struct {
	Capabilities []string `json:"capabilities"`
	SnapshotMode string   `json:"snapshotMode"`
}

type Repository interface {
	ListNodes() ([]nodes.Node, error)
	ListRuleSets() ([]rules.RuleSet, error)
	ListXRAYSnapshots(targetKind string, targetID string) ([]Snapshot, error)
	CreateXRAYSnapshot(input SnapshotInput) (Snapshot, error)
	GetXRAYSnapshot(id string) (Snapshot, error)
	ListXRAYProfiles() ([]Profile, error)
	CreateXRAYProfile(input CreateProfileInput) (Profile, error)
	UpdateXRAYProfile(id string, input CreateProfileInput) (Profile, error)
	DeleteXRAYProfile(id string) error
	GetXRAYProfile(id string) (Profile, error)
	QueueXRAYApplyTask(profile Profile, config string, summary string) (string, error)
	ListOpsResources(kind string) ([]ops.Resource, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Capabilities: []string{"render-config", "preview-diff", "snapshot-history", "apply-plan", "rollback"},
		SnapshotMode: "sqlite-and-filesystem",
	}
}

type Preview struct {
	Content string `json:"content"`
	Summary string `json:"summary"`
}

type Snapshot struct {
	ID         string `json:"id"`
	TargetKind string `json:"targetKind"`
	TargetID   string `json:"targetId"`
	Config     string `json:"config"`
	Summary    string `json:"summary"`
	CreatedAt  string `json:"createdAt"`
}

type SnapshotInput struct {
	TargetKind string `json:"targetKind"`
	TargetID   string `json:"targetId"`
	Config     string `json:"config"`
	Summary    string `json:"summary"`
}

type Profile struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	RemoteServerID string         `json:"remoteServerId"`
	RuntimeMode    string         `json:"runtimeMode"`
	BinaryPath     string         `json:"binaryPath"`
	ConfigPath     string         `json:"configPath"`
	ServiceName    string         `json:"serviceName"`
	Metadata       map[string]any `json:"metadata"`
	Enabled        bool           `json:"enabled"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt"`
}

type CreateProfileInput struct {
	Name           string         `json:"name"`
	RemoteServerID string         `json:"remoteServerId"`
	RuntimeMode    string         `json:"runtimeMode"`
	BinaryPath     string         `json:"binaryPath"`
	ConfigPath     string         `json:"configPath"`
	ServiceName    string         `json:"serviceName"`
	Metadata       map[string]any `json:"metadata"`
	Enabled        bool           `json:"enabled"`
}

type ApplyInput struct {
	ProfileID  string `json:"profileId"`
	TargetKind string `json:"targetKind"`
	TargetID   string `json:"targetId"`
	DryRun     bool   `json:"dryRun"`
}

type ApplyResult struct {
	Profile     Profile  `json:"profile"`
	Snapshot    Snapshot `json:"snapshot"`
	TaskID      string   `json:"taskId"`
	RuntimeMode string   `json:"runtimeMode"`
	Summary     string   `json:"summary"`
	Config      string   `json:"config"`
	DryRun      bool     `json:"dryRun"`
}

type xrayConfig struct {
	Log       map[string]string `json:"log"`
	Inbounds  []xrayInbound     `json:"inbounds"`
	Outbounds []xrayOutbound    `json:"outbounds"`
	Routing   xrayRouting       `json:"routing"`
}

type xrayInbound struct {
	Tag            string         `json:"tag"`
	Listen         string         `json:"listen"`
	Port           int            `json:"port"`
	Protocol       string         `json:"protocol"`
	Settings       map[string]any `json:"settings"`
	StreamSettings map[string]any `json:"streamSettings,omitempty"`
	Sniffing       map[string]any `json:"sniffing,omitempty"`
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
	inbounds, err := s.buildInbounds()
	if err != nil {
		return Preview{}, err
	}

	config := xrayConfig{
		Log:       map[string]string{"loglevel": "warning"},
		Inbounds:  inbounds,
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

func (s Service) ListSnapshots(targetKind string, targetID string) ([]Snapshot, error) {
	if s.repo == nil {
		return nil, errors.New("xray repository is not configured")
	}
	return s.repo.ListXRAYSnapshots(targetKind, targetID)
}

func (s Service) SaveSnapshot(targetKind string, targetID string) (Snapshot, error) {
	if s.repo == nil {
		return Snapshot{}, errors.New("xray repository is not configured")
	}
	preview, err := s.Preview()
	if err != nil {
		return Snapshot{}, err
	}
	return s.repo.CreateXRAYSnapshot(SnapshotInput{
		TargetKind: targetKind,
		TargetID:   targetID,
		Config:     preview.Content,
		Summary:    preview.Summary,
	})
}

func (s Service) RestoreSnapshot(id string) (Snapshot, error) {
	if s.repo == nil {
		return Snapshot{}, errors.New("xray repository is not configured")
	}
	if id == "" {
		return Snapshot{}, errors.New("xray snapshot id is required")
	}
	return s.repo.GetXRAYSnapshot(id)
}

func (s Service) ListProfiles() ([]Profile, error) {
	if s.repo == nil {
		return nil, errors.New("xray repository is not configured")
	}
	return s.repo.ListXRAYProfiles()
}

func (s Service) CreateProfile(input CreateProfileInput) (Profile, error) {
	if s.repo == nil {
		return Profile{}, errors.New("xray repository is not configured")
	}
	normalizeProfileInput(&input)
	if err := validateProfile(input); err != nil {
		return Profile{}, err
	}
	return s.repo.CreateXRAYProfile(input)
}

func (s Service) UpdateProfile(id string, input CreateProfileInput) (Profile, error) {
	if s.repo == nil {
		return Profile{}, errors.New("xray repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return Profile{}, errors.New("xray profile id is required")
	}
	normalizeProfileInput(&input)
	if err := validateProfile(input); err != nil {
		return Profile{}, err
	}
	return s.repo.UpdateXRAYProfile(id, input)
}

func (s Service) DeleteProfile(id string) error {
	if s.repo == nil {
		return errors.New("xray repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("xray profile id is required")
	}
	return s.repo.DeleteXRAYProfile(id)
}

func (s Service) Apply(input ApplyInput) (ApplyResult, error) {
	if s.repo == nil {
		return ApplyResult{}, errors.New("xray repository is not configured")
	}
	if strings.TrimSpace(input.ProfileID) == "" {
		return ApplyResult{}, errors.New("xray profile id is required")
	}
	profile, err := s.repo.GetXRAYProfile(input.ProfileID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !profile.Enabled {
		return ApplyResult{}, errors.New("xray profile is disabled")
	}
	preview, err := s.Preview()
	if err != nil {
		return ApplyResult{}, err
	}
	snapshot, err := s.repo.CreateXRAYSnapshot(SnapshotInput{
		TargetKind: fallbackString(input.TargetKind, "profile"),
		TargetID:   fallbackString(input.TargetID, profile.ID),
		Config:     preview.Content,
		Summary:    preview.Summary,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	result := ApplyResult{
		Profile:     profile,
		Snapshot:    snapshot,
		RuntimeMode: profile.RuntimeMode,
		Summary:     preview.Summary,
		Config:      preview.Content,
		DryRun:      input.DryRun,
	}
	if input.DryRun {
		return result, nil
	}
	if strings.TrimSpace(profile.RemoteServerID) == "" {
		return ApplyResult{}, errors.New("xray profile must be bound to a remote server before apply")
	}
	taskID, err := s.repo.QueueXRAYApplyTask(profile, preview.Content, preview.Summary)
	if err != nil {
		return ApplyResult{}, err
	}
	result.TaskID = taskID
	return result, nil
}

func (s Service) buildInbounds() ([]xrayInbound, error) {
	inbounds := []xrayInbound{
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
	}
	resources, err := s.repo.ListOpsResources("xray-inbound")
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		if !resource.Enabled {
			continue
		}
		inbounds = append(inbounds, inboundFromResource(resource))
	}
	return inbounds, nil
}

func inboundFromResource(resource ops.Resource) xrayInbound {
	config := resource.Config
	protocol := stringFromMap(config, "protocol", "vless")
	tag := stringFromMap(config, "tag", resource.Name)
	listen := stringFromMap(config, "listen", "0.0.0.0")
	port := intFromMap(config, "port", 443)
	network := stringFromMap(config, "network", "tcp")
	security := stringFromMap(config, "security", "none")
	inbound := xrayInbound{
		Tag:      tag,
		Listen:   listen,
		Port:     port,
		Protocol: protocol,
		Settings: inboundSettings(protocol, config),
		StreamSettings: map[string]any{
			"network":  network,
			"security": security,
		},
		Sniffing: map[string]any{
			"enabled":      boolFromMap(config, "sniffing", true),
			"destOverride": []string{"http", "tls", "quic"},
		},
	}
	if security == "reality" {
		inbound.StreamSettings["realitySettings"] = map[string]any{
			"show":        false,
			"dest":        stringFromMap(config, "realityDest", "www.microsoft.com:443"),
			"serverNames": stringSliceFromMap(config, "serverNames", []string{stringFromMap(config, "serverName", "www.microsoft.com")}),
			"privateKey":  stringFromMap(config, "privateKey", ""),
			"shortIds":    stringSliceFromMap(config, "shortIds", []string{""}),
		}
	}
	if security == "tls" {
		inbound.StreamSettings["tlsSettings"] = map[string]any{
			"serverName":   stringFromMap(config, "serverName", ""),
			"certificates": []map[string]any{{"certificateFile": stringFromMap(config, "certificateFile", ""), "keyFile": stringFromMap(config, "keyFile", "")}},
		}
	}
	return inbound
}

func inboundSettings(protocol string, config map[string]any) map[string]any {
	switch strings.ToLower(protocol) {
	case "vless":
		return map[string]any{
			"clients":    []map[string]any{{"id": stringFromMap(config, "uuid", ""), "flow": stringFromMap(config, "flow", "xtls-rprx-vision"), "email": stringFromMap(config, "email", "")}},
			"decryption": "none",
		}
	case "vmess":
		return map[string]any{"clients": []map[string]any{{"id": stringFromMap(config, "uuid", ""), "alterId": 0, "email": stringFromMap(config, "email", "")}}}
	case "trojan":
		return map[string]any{"clients": []map[string]any{{"password": stringFromMap(config, "password", ""), "email": stringFromMap(config, "email", "")}}}
	case "shadowsocks":
		return map[string]any{"method": stringFromMap(config, "method", "2022-blake3-aes-128-gcm"), "password": stringFromMap(config, "password", "")}
	default:
		return cloneAnyMap(config)
	}
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

func normalizeProfileInput(input *CreateProfileInput) {
	input.Name = strings.TrimSpace(input.Name)
	input.RemoteServerID = strings.TrimSpace(input.RemoteServerID)
	input.RuntimeMode = strings.TrimSpace(input.RuntimeMode)
	input.BinaryPath = strings.TrimSpace(input.BinaryPath)
	input.ConfigPath = strings.TrimSpace(input.ConfigPath)
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	if input.RuntimeMode == "" {
		input.RuntimeMode = "external"
	}
	if input.BinaryPath == "" {
		input.BinaryPath = "xray"
	}
	if input.ConfigPath == "" {
		input.ConfigPath = "/usr/local/etc/xray/config.json"
	}
	if input.ServiceName == "" {
		input.ServiceName = "xray"
	}
}

func validateProfile(input CreateProfileInput) error {
	if input.Name == "" {
		return errors.New("xray profile name is required")
	}
	switch input.RuntimeMode {
	case "external", "inline":
		return nil
	default:
		return errors.New("xray runtime mode must be external or inline")
	}
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
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

func stringFromMap(values map[string]any, key string, fallback string) string {
	if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func intFromMap(values map[string]any, key string, fallback int) int {
	switch value := values[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return fallback
	}
}

func boolFromMap(values map[string]any, key string, fallback bool) bool {
	if value, ok := values[key].(bool); ok {
		return value
	}
	return fallback
}

func stringSliceFromMap(values map[string]any, key string, fallback []string) []string {
	raw, ok := values[key]
	if !ok {
		return fallback
	}
	switch value := raw.(type) {
	case []string:
		return value
	case []any:
		var items []string
		for _, item := range value {
			if text, ok := item.(string); ok {
				items = append(items, text)
			}
		}
		if len(items) > 0 {
			return items
		}
	case string:
		if strings.TrimSpace(value) != "" {
			return []string{strings.TrimSpace(value)}
		}
	}
	return fallback
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
