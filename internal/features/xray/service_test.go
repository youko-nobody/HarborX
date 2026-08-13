package xray

import (
	"encoding/json"
	"strings"
	"testing"

	"harborx/internal/features/nodes"
	"harborx/internal/features/ops"
	"harborx/internal/features/rules"
)

type fakeRepo struct {
	nodes    []nodes.Node
	ruleSets []rules.RuleSet
	resources []ops.Resource
}

func (f *fakeRepo) ListNodes() ([]nodes.Node, error)                  { return f.nodes, nil }
func (f *fakeRepo) ListRuleSets() ([]rules.RuleSet, error)            { return f.ruleSets, nil }
func (f *fakeRepo) ListXRAYSnapshots(targetKind string, targetID string) ([]Snapshot, error) {
	return nil, nil
}
func (f *fakeRepo) CreateXRAYSnapshot(input SnapshotInput) (Snapshot, error) {
	return Snapshot{}, nil
}
func (f *fakeRepo) GetXRAYSnapshot(id string) (Snapshot, error) { return Snapshot{}, nil }
func (f *fakeRepo) ListXRAYProfiles() ([]Profile, error)        { return nil, nil }
func (f *fakeRepo) CreateXRAYProfile(input CreateProfileInput) (Profile, error) {
	return Profile{}, nil
}
func (f *fakeRepo) UpdateXRAYProfile(id string, input CreateProfileInput) (Profile, error) {
	return Profile{}, nil
}
func (f *fakeRepo) DeleteXRAYProfile(id string) error { return nil }
func (f *fakeRepo) GetXRAYProfile(id string) (Profile, error) {
	return Profile{}, nil
}
func (f *fakeRepo) QueueXRAYApplyTask(profile Profile, config string, summary string) (string, error) {
	return "task-1", nil
}
func (f *fakeRepo) ListOpsResources(kind string) ([]ops.Resource, error) {
	if kind == "xray-inbound" {
		return f.resources, nil
	}
	return nil, nil
}

func TestPreviewGeneratesValidConfig(t *testing.T) {
	repo := &fakeRepo{
		nodes: []nodes.Node{
			{Name: "Tokyo", Protocol: "vless", ServerHost: "203.0.113.10", ServerPort: 443, Enabled: true, Metadata: map[string]any{"user": "uuid"}},
			{Name: "Offline Node", Protocol: "vmess", ServerHost: "203.0.113.99", ServerPort: 80, Enabled: false},
		},
		resources: []ops.Resource{
			{Name: "reality-in", Enabled: true, Config: map[string]any{
				"protocol": "vless", "port": 443, "network": "tcp", "security": "reality",
				"uuid": "abc-123", "serverName": "www.microsoft.com", "privateKey": "key-1",
			}},
		},
	}
	svc := NewService(repo)

	preview, err := svc.Preview()
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if preview.Content == "" {
		t.Fatal("Preview returned empty content")
	}
	if !strings.Contains(preview.Summary, "3 inbounds") {
		t.Fatalf("unexpected summary: %q", preview.Summary)
	}

	var config xrayConfig
	if err := json.Unmarshal([]byte(preview.Content), &config); err != nil {
		t.Fatalf("preview content is not valid JSON: %v", err)
	}
	// Default socks/http inbounds + 1 reality inbound.
	if len(config.Inbounds) != 3 {
		t.Fatalf("expected 3 inbounds, got %d", len(config.Inbounds))
	}
	// direct + block + only enabled nodes.
	if len(config.Outbounds) != 3 {
		t.Fatalf("expected 3 outbounds, got %d", len(config.Outbounds))
	}
	// Disabled nodes must never appear in generated config.
	for _, ob := range config.Outbounds {
		if ob.Tag == "Offline Node" {
			t.Fatal("disabled node leaked into outbounds")
		}
	}
	foundReality := false
	for _, inbound := range config.Inbounds {
		if inbound.Protocol == "vless" && inbound.StreamSettings["security"] == "reality" {
			foundReality = true
			reality := inbound.StreamSettings["realitySettings"].(map[string]any)
			if reality["privateKey"] != "key-1" {
				t.Fatalf("reality private key not preserved: %v", reality)
			}
		}
	}
	if !foundReality {
		t.Fatal("reality inbound missing from config")
	}
}

func TestPreviewNilRepo(t *testing.T) {
	var svc Service
	if _, err := svc.Preview(); err == nil {
		t.Fatal("nil repo must return error")
	}
}

func TestPreviewEmptyState(t *testing.T) {
	svc := NewService(&fakeRepo{})
	preview, err := svc.Preview()
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	var config xrayConfig
	if err := json.Unmarshal([]byte(preview.Content), &config); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(config.Outbounds) != 2 {
		t.Fatalf("expected direct+block outbounds only, got %d", len(config.Outbounds))
	}
}
