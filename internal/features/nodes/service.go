package nodes

import "errors"

type Node struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	SourceKind string         `json:"sourceKind"`
	Protocol   string         `json:"protocol"`
	ServerHost string         `json:"serverHost"`
	ServerPort int            `json:"serverPort"`
	Tags       []string       `json:"tags"`
	Metadata   map[string]any `json:"metadata"`
	Enabled    bool           `json:"enabled"`
	CreatedAt  string         `json:"createdAt"`
	UpdatedAt  string         `json:"updatedAt"`
}

type CreateInput struct {
	Name       string         `json:"name"`
	SourceKind string         `json:"sourceKind"`
	Protocol   string         `json:"protocol"`
	ServerHost string         `json:"serverHost"`
	ServerPort int            `json:"serverPort"`
	Tags       []string       `json:"tags"`
	Metadata   map[string]any `json:"metadata"`
	Enabled    bool           `json:"enabled"`
}

type Summary struct {
	SupportedSources   []string `json:"supportedSources"`
	SupportedProtocols []string `json:"supportedProtocols"`
	Capabilities       []string `json:"capabilities"`
}

type Repository interface {
	ListNodes() ([]Node, error)
	CreateNode(input CreateInput) (Node, error)
	DeleteNode(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		SupportedSources:   []string{"manual", "subscription-import", "remote-sync"},
		SupportedProtocols: []string{"vmess", "vless", "trojan", "shadowsocks", "hysteria2", "tuic", "snell", "socks5"},
		Capabilities:       []string{"crud", "tagging", "health-check", "bind-to-template"},
	}
}

func (s Service) List() ([]Node, error) {
	if s.repo == nil {
		return nil, errors.New("nodes repository is not configured")
	}
	return s.repo.ListNodes()
}

func (s Service) Create(input CreateInput) (Node, error) {
	if s.repo == nil {
		return Node{}, errors.New("nodes repository is not configured")
	}
	return s.repo.CreateNode(input)
}

func (s Service) Delete(id string) error {
	if s.repo == nil {
		return errors.New("nodes repository is not configured")
	}
	return s.repo.DeleteNode(id)
}
