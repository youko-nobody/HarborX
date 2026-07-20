package certificates

import (
	"errors"
	"strings"
)

type Summary struct {
	Providers      []string `json:"providers"`
	Capabilities   []string `json:"capabilities"`
	DeploymentMode string   `json:"deploymentMode"`
}

type Certificate struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	ProviderID string `json:"providerId"`
	CertPEM    string `json:"certPem"`
	KeyPEM     string `json:"keyPem"`
	AutoRenew  bool   `json:"autoRenew"`
	AutoDeploy bool   `json:"autoDeploy"`
	ExpiresAt  string `json:"expiresAt"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type CreateInput struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	ProviderID string `json:"providerId"`
	CertPEM    string `json:"certPem"`
	KeyPEM     string `json:"keyPem"`
	AutoRenew  bool   `json:"autoRenew"`
	AutoDeploy bool   `json:"autoDeploy"`
	ExpiresAt  string `json:"expiresAt"`
}

type Repository interface {
	ListCertificates() ([]Certificate, error)
	CreateCertificate(input CreateInput) (Certificate, error)
	UpdateCertificate(id string, input CreateInput) (Certificate, error)
	DeleteCertificate(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Providers:      []string{"cloudflare", "alidns", "dnspod", "tencent", "godaddy", "namesilo", "webroot"},
		Capabilities:   []string{"issue", "renew", "deploy", "upload", "auto-renew"},
		DeploymentMode: "master-and-remote",
	}
}

func (s Service) List() ([]Certificate, error) {
	if s.repo == nil {
		return nil, errors.New("certificate repository is not configured")
	}
	return s.repo.ListCertificates()
}

func (s Service) Create(input CreateInput) (Certificate, error) {
	if s.repo == nil {
		return Certificate{}, errors.New("certificate repository is not configured")
	}
	if err := validate(input); err != nil {
		return Certificate{}, err
	}
	return s.repo.CreateCertificate(input)
}

func (s Service) Update(id string, input CreateInput) (Certificate, error) {
	if s.repo == nil {
		return Certificate{}, errors.New("certificate repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return Certificate{}, errors.New("certificate id is required")
	}
	if err := validate(input); err != nil {
		return Certificate{}, err
	}
	return s.repo.UpdateCertificate(id, input)
}

func (s Service) Delete(id string) error {
	if s.repo == nil {
		return errors.New("certificate repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("certificate id is required")
	}
	return s.repo.DeleteCertificate(id)
}

func validate(input CreateInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("certificate name is required")
	}
	if strings.TrimSpace(input.Domain) == "" {
		return errors.New("certificate domain is required")
	}
	return nil
}
