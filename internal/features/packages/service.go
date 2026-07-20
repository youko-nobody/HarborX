package packages

import (
	"errors"
	"strings"
)

type Summary struct {
	Capabilities []string `json:"capabilities"`
}

type Package struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	BandwidthBytes int64    `json:"bandwidthBytes"`
	DeviceLimit    int      `json:"deviceLimit"`
	DurationDays   int      `json:"durationDays"`
	Features       []string `json:"features"`
	Enabled        bool     `json:"enabled"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

type CreatePackageInput struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	BandwidthBytes int64    `json:"bandwidthBytes"`
	DeviceLimit    int      `json:"deviceLimit"`
	DurationDays   int      `json:"durationDays"`
	Features       []string `json:"features"`
	Enabled        bool     `json:"enabled"`
}

type Entitlement struct {
	ID        string         `json:"id"`
	UserID    string         `json:"userId"`
	PackageID string         `json:"packageId"`
	Status    string         `json:"status"`
	StartedAt string         `json:"startedAt"`
	ExpiresAt string         `json:"expiresAt"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
}

type CreateEntitlementInput struct {
	UserID    string         `json:"userId"`
	PackageID string         `json:"packageId"`
	Status    string         `json:"status"`
	ExpiresAt string         `json:"expiresAt"`
	Metadata  map[string]any `json:"metadata"`
}

type Repository interface {
	ListPackages() ([]Package, error)
	CreatePackage(input CreatePackageInput) (Package, error)
	UpdatePackage(id string, input CreatePackageInput) (Package, error)
	DeletePackage(id string) error
	ListEntitlements(userID string) ([]Entitlement, error)
	CreateEntitlement(input CreateEntitlementInput) (Entitlement, error)
	UpdateEntitlement(id string, input CreateEntitlementInput) (Entitlement, error)
	DeleteEntitlement(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Capabilities: []string{"package-crud", "user-entitlements", "traffic-limits", "device-limits", "no-license-gating"},
	}
}

func (s Service) ListPackages() ([]Package, error) {
	if s.repo == nil {
		return nil, errors.New("packages repository is not configured")
	}
	return s.repo.ListPackages()
}

func (s Service) CreatePackage(input CreatePackageInput) (Package, error) {
	if s.repo == nil {
		return Package{}, errors.New("packages repository is not configured")
	}
	if err := validatePackage(input); err != nil {
		return Package{}, err
	}
	return s.repo.CreatePackage(input)
}

func (s Service) UpdatePackage(id string, input CreatePackageInput) (Package, error) {
	if s.repo == nil {
		return Package{}, errors.New("packages repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return Package{}, errors.New("package id is required")
	}
	if err := validatePackage(input); err != nil {
		return Package{}, err
	}
	return s.repo.UpdatePackage(id, input)
}

func (s Service) DeletePackage(id string) error {
	if s.repo == nil {
		return errors.New("packages repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("package id is required")
	}
	return s.repo.DeletePackage(id)
}

func (s Service) ListEntitlements(userID string) ([]Entitlement, error) {
	if s.repo == nil {
		return nil, errors.New("packages repository is not configured")
	}
	return s.repo.ListEntitlements(userID)
}

func (s Service) CreateEntitlement(input CreateEntitlementInput) (Entitlement, error) {
	if s.repo == nil {
		return Entitlement{}, errors.New("packages repository is not configured")
	}
	if err := validateEntitlement(input); err != nil {
		return Entitlement{}, err
	}
	return s.repo.CreateEntitlement(input)
}

func (s Service) UpdateEntitlement(id string, input CreateEntitlementInput) (Entitlement, error) {
	if s.repo == nil {
		return Entitlement{}, errors.New("packages repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return Entitlement{}, errors.New("entitlement id is required")
	}
	if err := validateEntitlement(input); err != nil {
		return Entitlement{}, err
	}
	return s.repo.UpdateEntitlement(id, input)
}

func (s Service) DeleteEntitlement(id string) error {
	if s.repo == nil {
		return errors.New("packages repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("entitlement id is required")
	}
	return s.repo.DeleteEntitlement(id)
}

func validatePackage(input CreatePackageInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("package name is required")
	}
	if input.BandwidthBytes < 0 {
		return errors.New("package bandwidth cannot be negative")
	}
	if input.DeviceLimit < 0 {
		return errors.New("package device limit cannot be negative")
	}
	if input.DurationDays <= 0 {
		return errors.New("package duration days must be greater than 0")
	}
	return nil
}

func validateEntitlement(input CreateEntitlementInput) error {
	if strings.TrimSpace(input.UserID) == "" {
		return errors.New("entitlement user id is required")
	}
	if strings.TrimSpace(input.PackageID) == "" {
		return errors.New("entitlement package id is required")
	}
	if strings.TrimSpace(input.Status) == "" {
		return errors.New("entitlement status is required")
	}
	switch input.Status {
	case "active", "paused", "expired", "cancelled":
		return nil
	default:
		return errors.New("unsupported entitlement status")
	}
}
