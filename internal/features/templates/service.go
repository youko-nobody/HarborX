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
	UpdateTemplate(id string, input CreateInput) (Template, error)
	DeleteTemplate(id string) error
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

func (s Service) Update(id string, input CreateInput) (Template, error) {
	if s.repo == nil {
		return Template{}, errors.New("templates repository is not configured")
	}
	if id == "" {
		return Template{}, errors.New("template id is required")
	}
	return s.repo.UpdateTemplate(id, input)
}

func (s Service) Delete(id string) error {
	if s.repo == nil {
		return errors.New("templates repository is not configured")
	}
	if id == "" {
		return errors.New("template id is required")
	}
	return s.repo.DeleteTemplate(id)
}
