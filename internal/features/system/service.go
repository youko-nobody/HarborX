package system

import (
	"errors"
	"strings"
)

type Summary struct {
	Capabilities []string `json:"capabilities"`
	DefaultTheme string   `json:"defaultTheme"`
}

type Setting struct {
	Key       string         `json:"key"`
	Value     map[string]any `json:"value"`
	UpdatedAt string         `json:"updatedAt"`
}

type UpsertSettingInput struct {
	Value map[string]any `json:"value"`
}

type Repository interface {
	ListSystemSettings() ([]Setting, error)
	UpsertSystemSetting(key string, input UpsertSettingInput) (Setting, error)
	DeleteSystemSetting(key string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Capabilities: []string{
			"theme-settings",
			"dashboard-refresh",
			"default-template",
			"short-link-behavior",
			"operator-permissions",
		},
		DefaultTheme: "sand",
	}
}

func (s Service) ListSettings() ([]Setting, error) {
	if s.repo == nil {
		return nil, errors.New("system repository is not configured")
	}
	return s.repo.ListSystemSettings()
}

func (s Service) UpsertSetting(key string, input UpsertSettingInput) (Setting, error) {
	if s.repo == nil {
		return Setting{}, errors.New("system repository is not configured")
	}
	if strings.TrimSpace(key) == "" {
		return Setting{}, errors.New("system setting key is required")
	}
	return s.repo.UpsertSystemSetting(key, input)
}

func (s Service) DeleteSetting(key string) error {
	if s.repo == nil {
		return errors.New("system repository is not configured")
	}
	if strings.TrimSpace(key) == "" {
		return errors.New("system setting key is required")
	}
	return s.repo.DeleteSystemSetting(key)
}
