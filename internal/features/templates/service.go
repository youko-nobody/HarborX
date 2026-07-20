package templates

import "errors"

type Template struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	Variables   []string `json:"variables"`
	Content     string   `json:"content"`
	Locked      bool     `json:"locked"`
}

type CreateInput struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	Variables   []string `json:"variables"`
	Content     string   `json:"content"`
}

type Repository interface {
	ListTemplates() ([]Template, error)
	CreateTemplate(input CreateInput) (Template, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) List() ([]Template, error) {
	if s.repo == nil {
		return nil, errors.New("templates repository is not configured")
	}
	return s.repo.ListTemplates()
}

func (s Service) Create(input CreateInput) (Template, error) {
	if s.repo == nil {
		return Template{}, errors.New("templates repository is not configured")
	}
	return s.repo.CreateTemplate(input)
}
