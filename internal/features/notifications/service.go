package notifications

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	FindNotificationChannel(id string) (Channel, error)
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

func (s Service) TestChannel(id string, message string) error {
	if s.repo == nil {
		return errors.New("notification repository is not configured")
	}
	channel, err := s.repo.FindNotificationChannel(id)
	if err != nil {
		return err
	}
	if !channel.Enabled {
		return errors.New("notification channel is disabled")
	}
	if strings.TrimSpace(message) == "" {
		message = "HarborX notification test"
	}

	switch channel.ChannelKind {
	case "telegram":
		return testTelegram(channel.Config, message)
	case "webhook":
		return testWebhook(channel.Config, message)
	default:
		return errors.New("test delivery is not implemented for this channel kind")
	}
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

func testTelegram(config map[string]any, message string) error {
	botToken, _ := config["botToken"].(string)
	chatID, _ := config["chatId"].(string)
	if strings.TrimSpace(botToken) == "" || strings.TrimSpace(chatID) == "" {
		return errors.New("telegram channel requires botToken and chatId")
	}

	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", message)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram request failed: %s", resp.Status)
	}
	return nil
}

func testWebhook(config map[string]any, message string) error {
	endpoint, _ := config["url"].(string)
	if strings.TrimSpace(endpoint) == "" {
		return errors.New("webhook channel requires url")
	}
	body, _ := json.Marshal(map[string]any{"message": message, "source": "harborx"})
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook request failed: %s", resp.Status)
	}
	return nil
}
