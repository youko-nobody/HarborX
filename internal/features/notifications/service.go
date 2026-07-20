package notifications

import (
	"errors"
	"strings"
)

type Summary struct {
	Channels     []string `json:"channels"`
	Capabilities []string `json:"capabilities"`
}

type Channel struct {
	ID          string         `json:"id"`
	ChannelKind string         `json:"channelKind"`
	Name        string         `json:"name"`
	Config      map[string]any `json:"config"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
}

type CreateInput struct {
	ChannelKind string         `json:"channelKind"`
	Name        string         `json:"name"`
	Config      map[string]any `json:"config"`
	Enabled     bool           `json:"enabled"`
}

type Repository interface {
	ListNotificationChannels() ([]Channel, error)
	CreateNotificationChannel(input CreateInput) (Channel, error)
	UpdateNotificationChannel(id string, input CreateInput) (Channel, error)
	DeleteNotificationChannel(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Channels:     []string{"telegram"},
		Capabilities: []string{"alerts", "daily-summary", "traffic-thresholds", "server-status"},
	}
}

func (s Service) ListChannels() ([]Channel, error) {
	if s.repo == nil {
		return nil, errors.New("notification repository is not configured")
	}
	return s.repo.ListNotificationChannels()
}

func (s Service) CreateChannel(input CreateInput) (Channel, error) {
	if s.repo == nil {
		return Channel{}, errors.New("notification repository is not configured")
	}
	if err := validate(input); err != nil {
		return Channel{}, err
	}
	return s.repo.CreateNotificationChannel(input)
}

func (s Service) UpdateChannel(id string, input CreateInput) (Channel, error) {
	if s.repo == nil {
		return Channel{}, errors.New("notification repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return Channel{}, errors.New("notification channel id is required")
	}
	if err := validate(input); err != nil {
		return Channel{}, err
	}
	return s.repo.UpdateNotificationChannel(id, input)
}

func (s Service) DeleteChannel(id string) error {
	if s.repo == nil {
		return errors.New("notification repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("notification channel id is required")
	}
	return s.repo.DeleteNotificationChannel(id)
}

func validate(input CreateInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("notification channel name is required")
	}
	if strings.TrimSpace(input.ChannelKind) == "" {
		return errors.New("notification channel kind is required")
	}
	switch input.ChannelKind {
	case "telegram", "webhook", "email":
		return nil
	default:
		return errors.New("unsupported notification channel kind")
	}
}
