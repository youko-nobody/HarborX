package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

type Summary struct {
	LoginModes        []string `json:"loginModes"`
	SessionStore      string   `json:"sessionStore"`
	SupportsTOTP      bool     `json:"supportsTotp"`
	SupportsAPITokens bool     `json:"supportsApiTokens"`
}

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Role         string
	Status       string
	DisplayName  string
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type Repository interface {
	GetUserByUsername(username string) (User, error)
	UpdateUserPasswordHash(userID string, passwordHash string) error
	CreateAPIToken(userID string, name string, tokenHash string) error
	FindAPITokenByHash(tokenHash string) (User, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	service := Service{repo: repo}
	service.ensureAdminPassword()
	return service
}

func (Service) Summary() Summary {
	return Summary{
		LoginModes:        []string{"password", "api-token"},
		SessionStore:      "sqlite",
		SupportsTOTP:      true,
		SupportsAPITokens: true,
	}
}

func (s Service) Login(input LoginInput) (LoginResponse, error) {
	if s.repo == nil {
		return LoginResponse{}, errors.New("auth repository is not configured")
	}
	user, err := s.repo.GetUserByUsername(strings.TrimSpace(input.Username))
	if err != nil {
		return LoginResponse{}, errors.New("invalid username or password")
	}
	if user.Status != "active" {
		return LoginResponse{}, errors.New("user is disabled")
	}
	if !VerifyPassword(input.Password, user.PasswordHash) {
		return LoginResponse{}, errors.New("invalid username or password")
	}

	token, tokenHash, err := newToken()
	if err != nil {
		return LoginResponse{}, err
	}
	if err := s.repo.CreateAPIToken(user.ID, "web-session", tokenHash); err != nil {
		return LoginResponse{}, err
	}

	user.PasswordHash = ""
	return LoginResponse{Token: token, User: user}, nil
}

func (s Service) AuthenticateBearer(header string) (User, error) {
	if s.repo == nil {
		return User{}, errors.New("auth repository is not configured")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return User{}, errors.New("missing bearer token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return User{}, errors.New("missing bearer token")
	}
	return s.repo.FindAPITokenByHash(HashToken(token))
}

func (s Service) ensureAdminPassword() {
	if s.repo == nil {
		return
	}
	admin, err := s.repo.GetUserByUsername("admin")
	if err != nil || admin.PasswordHash != "" {
		return
	}

	password := os.Getenv("HARBORX_ADMIN_PASSWORD")
	if password == "" {
		password = randomHumanPassword()
		log.Printf("HarborX bootstrap admin password: %s", password)
	}
	hash, err := HashPassword(password)
	if err != nil {
		log.Printf("failed to hash bootstrap admin password: %v", err)
		return
	}
	if err := s.repo.UpdateUserPasswordHash(admin.ID, hash); err != nil {
		log.Printf("failed to store bootstrap admin password: %v", err)
	}
}

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password is required")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := pbkdf2.Key([]byte(password), salt, 210000, 32, sha256.New)
	return fmt.Sprintf("pbkdf2$210000$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived)), nil
}

func VerifyPassword(password string, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false
	}
	var iterations int
	if _, err := fmt.Sscanf(parts[1], "%d", &iterations); err != nil || iterations <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	actual := pbkdf2.Key([]byte(password), salt, iterations, len(expected), sha256.New)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func newToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := "hx_" + base64.RawURLEncoding.EncodeToString(raw)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

func randomHumanPassword() string {
	raw := make([]byte, 9)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("harborx-%d", time.Now().Unix())
	}
	return "harborx-" + base64.RawURLEncoding.EncodeToString(raw)
}
