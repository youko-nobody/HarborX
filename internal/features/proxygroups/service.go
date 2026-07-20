package proxygroups

import (
	"errors"
	"strings"
)

type Summary struct {
	GroupKinds   []string `json:"groupKinds"`
	Capabilities []string `json:"capabilities"`
}

type ProxyGroup struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	GroupKind string         `json:"groupKind"`
	Config    map[string]any `json:"config"`
	SortOrder int            `json:"sortOrder"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
}

type CreateInput struct {
	Name      string         `json:"name"`
	GroupKind string         `json:"groupKind"`
	Config    map[string]any `json:"config"`
	SortOrder int            `json:"sortOrder"`
}

type Repository interface {
	ListProxyGroups() ([]ProxyGroup, error)
	CreateProxyGroup(input CreateInput) (ProxyGroup, error)
	UpdateProxyGroup(id string, input CreateInput) (ProxyGroup, error)
	DeleteProxyGroup(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		GroupKinds:   []string{"select", "url-test", "fallback", "load-balance", "relay"},
		Capabilities: []string{"crud", "ordering", "policy-binding"},
	}
}

func (s Service) List() ([]ProxyGroup, error) {
	if s.repo == nil {
		return nil, errors.New("proxy group repository is not configured")
	}
	return s.repo.ListProxyGroups()
}

func (s Service) Create(input CreateInput) (ProxyGroup, error) {
	if s.repo == nil {
		return ProxyGroup{}, errors.New("proxy group repository is not configured")
	}
	if err := validate(input); err != nil {
		return ProxyGroup{}, err
	}
	return s.repo.CreateProxyGroup(input)
}

func (s Service) Update(id string, input CreateInput) (ProxyGroup, error) {
	if s.repo == nil {
		return ProxyGroup{}, errors.New("proxy group repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return ProxyGroup{}, errors.New("proxy group id is required")
	}
	if err := validate(input); err != nil {
		return ProxyGroup{}, err
	}
	return s.repo.UpdateProxyGroup(id, input)
}

func (s Service) Delete(id string) error {
	if s.repo == nil {
		return errors.New("proxy group repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("proxy group id is required")
	}
	return s.repo.DeleteProxyGroup(id)
}

func validate(input CreateInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("proxy group name is required")
	}
	if strings.TrimSpace(input.GroupKind) == "" {
		return errors.New("proxy group kind is required")
	}
	switch input.GroupKind {
	case "select", "url-test", "fallback", "load-balance", "relay":
		return nil
	default:
		return errors.New("unsupported proxy group kind")
	}
}
