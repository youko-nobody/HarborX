package dns

import (
	"errors"
	"strings"
)

type Summary struct {
	Providers    []string `json:"providers"`
	Capabilities []string `json:"capabilities"`
}

type Provider struct {
	ID           string         `json:"id"`
	ProviderKind string         `json:"providerKind"`
	Name         string         `json:"name"`
	Credentials  map[string]any `json:"credentials"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

type CreateProviderInput struct {
	ProviderKind string         `json:"providerKind"`
	Name         string         `json:"name"`
	Credentials  map[string]any `json:"credentials"`
}

type Repository interface {
	ListDNSProviders() ([]Provider, error)
	CreateDNSProvider(input CreateProviderInput) (Provider, error)
	UpdateDNSProvider(id string, input CreateProviderInput) (Provider, error)
	DeleteDNSProvider(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Providers:    []string{"cloudflare", "alidns", "dnspod", "tencent", "godaddy", "namesilo"},
		Capabilities: []string{"record-manage", "zone-lookup", "acme-support", "dynamic-dns"},
	}
}

func (s Service) ListProviders() ([]Provider, error) {
	if s.repo == nil {
		return nil, errors.New("dns repository is not configured")
	}
	return s.repo.ListDNSProviders()
}

func (s Service) CreateProvider(input CreateProviderInput) (Provider, error) {
	if s.repo == nil {
		return Provider{}, errors.New("dns repository is not configured")
	}
	if err := validateProvider(input); err != nil {
		return Provider{}, err
	}
	return s.repo.CreateDNSProvider(input)
}

func (s Service) UpdateProvider(id string, input CreateProviderInput) (Provider, error) {
	if s.repo == nil {
		return Provider{}, errors.New("dns repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return Provider{}, errors.New("dns provider id is required")
	}
	if err := validateProvider(input); err != nil {
		return Provider{}, err
	}
	return s.repo.UpdateDNSProvider(id, input)
}

func (s Service) DeleteProvider(id string) error {
	if s.repo == nil {
		return errors.New("dns repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("dns provider id is required")
	}
	return s.repo.DeleteDNSProvider(id)
}

func validateProvider(input CreateProviderInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("dns provider name is required")
	}
	if strings.TrimSpace(input.ProviderKind) == "" {
		return errors.New("dns provider kind is required")
	}
	switch input.ProviderKind {
	case "cloudflare", "alidns", "dnspod", "tencent", "godaddy", "namesilo":
		return nil
	default:
		return errors.New("unsupported dns provider kind")
	}
}
