package subscriptions

import "errors"

type Subscription struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	OwnerUserID  string         `json:"ownerUserId"`
	OutputFormat string         `json:"outputFormat"`
	TemplateID   string         `json:"templateId"`
	Sources      []string       `json:"sources"`
	Options      map[string]any `json:"options"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

type CreateInput struct {
	Name         string         `json:"name"`
	OwnerUserID  string         `json:"ownerUserId"`
	OutputFormat string         `json:"outputFormat"`
	TemplateID   string         `json:"templateId"`
	Sources      []string       `json:"sources"`
	Options      map[string]any `json:"options"`
}

type Summary struct {
	OutputFormats []string `json:"outputFormats"`
	Capabilities  []string `json:"capabilities"`
}

type Repository interface {
	ListSubscriptions() ([]Subscription, error)
	CreateSubscription(input CreateInput) (Subscription, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		OutputFormats: []string{
			"clash-meta",
			"surge",
			"loon",
			"quantumult-x",
			"shadowrocket",
			"sing-box",
			"stash",
			"surfboard",
			"v2ray",
		},
		Capabilities: []string{"per-user-subscribe", "template-render", "short-links", "merge-sources"},
	}
}

func (s Service) List() ([]Subscription, error) {
	if s.repo == nil {
		return nil, errors.New("subscriptions repository is not configured")
	}
	return s.repo.ListSubscriptions()
}

func (s Service) Create(input CreateInput) (Subscription, error) {
	if s.repo == nil {
		return Subscription{}, errors.New("subscriptions repository is not configured")
	}
	return s.repo.CreateSubscription(input)
}
