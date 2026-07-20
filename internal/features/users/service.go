package users

import (
	"errors"
	"strings"

	"harborx/internal/features/auth"
)

type Summary struct {
	Roles          []string `json:"roles"`
	SupportsLimits bool     `json:"supportsLimits"`
	SupportsPrefs  bool     `json:"supportsPrefs"`
}

type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type CreateInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

type UpdateInput struct {
	Role        string `json:"role"`
	Status      string `json:"status"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type Repository interface {
	ListUsers() ([]User, error)
	CreateUser(input CreateInput, passwordHash string) (User, error)
	UpdateUser(id string, input UpdateInput, passwordHash string) (User, error)
	DeleteUser(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Roles:          []string{"admin", "member"},
		SupportsLimits: true,
		SupportsPrefs:  true,
	}
}

func (s Service) List() ([]User, error) {
	if s.repo == nil {
		return nil, errors.New("users repository is not configured")
	}
	return s.repo.ListUsers()
}

func (s Service) Create(input CreateInput) (User, error) {
	if s.repo == nil {
		return User{}, errors.New("users repository is not configured")
	}
	if err := validateCreate(input); err != nil {
		return User{}, err
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return User{}, err
	}
	return s.repo.CreateUser(input, passwordHash)
}

func (s Service) Update(id string, input UpdateInput) (User, error) {
	if s.repo == nil {
		return User{}, errors.New("users repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return User{}, errors.New("user id is required")
	}
	if input.Role == "" {
		input.Role = "member"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if !supportedRole(input.Role) {
		return User{}, errors.New("unsupported user role")
	}
	if !supportedStatus(input.Status) {
		return User{}, errors.New("unsupported user status")
	}
	passwordHash := ""
	if strings.TrimSpace(input.Password) != "" {
		var err error
		passwordHash, err = auth.HashPassword(input.Password)
		if err != nil {
			return User{}, err
		}
	}
	return s.repo.UpdateUser(id, input, passwordHash)
}

func (s Service) Delete(id string) error {
	if s.repo == nil {
		return errors.New("users repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("user id is required")
	}
	if id == "local-admin" {
		return errors.New("local admin cannot be deleted")
	}
	return s.repo.DeleteUser(id)
}

func validateCreate(input CreateInput) error {
	if strings.TrimSpace(input.Username) == "" {
		return errors.New("username is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return errors.New("password is required")
	}
	if input.Role == "" {
		input.Role = "member"
	}
	if !supportedRole(input.Role) {
		return errors.New("unsupported user role")
	}
	return nil
}

func supportedRole(value string) bool {
	switch value {
	case "admin", "member":
		return true
	default:
		return false
	}
}

func supportedStatus(value string) bool {
	switch value {
	case "active", "disabled":
		return true
	default:
		return false
	}
}
