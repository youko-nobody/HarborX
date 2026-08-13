package audit

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Entry is a single auditable action recorded by the control plane.
type Entry struct {
	ID            string         `json:"id"`
	ActorID       string         `json:"actorId"`
	ActorUsername string         `json:"actorUsername"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resourceType"`
	ResourceID    string         `json:"resourceId"`
	Detail        map[string]any `json:"detail,omitempty"`
	IP            string         `json:"ip"`
	CreatedAt     string         `json:"createdAt"`
}

type CreateEntryInput struct {
	ActorID       string         `json:"actorId"`
	ActorUsername string         `json:"actorUsername"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resourceType"`
	ResourceID    string         `json:"resourceId"`
	Detail        map[string]any `json:"detail,omitempty"`
	IP            string         `json:"ip"`
}

type Summary struct {
	Enabled  bool   `json:"enabled"`
	Retention string `json:"retention"`
}

type Repository interface {
	CreateAuditEntry(entry Entry) error
	ListAuditEntries(limit int) ([]Entry, error)
	DeleteAuditEntriesBefore(cutoff time.Time) (int64, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{Enabled: true, Retention: "90d"}
}

func (s Service) Record(input CreateEntryInput) error {
	if s.repo == nil {
		return errors.New("audit repository is not configured")
	}
	if strings.TrimSpace(input.Action) == "" {
		return errors.New("audit action is required")
	}
	if strings.TrimSpace(input.ActorID) == "" {
		input.ActorID = "system"
	}
	entry := Entry{
		ID:            fmt.Sprintf("audit-%s", uuid.NewString()),
		ActorID:       input.ActorID,
		ActorUsername: input.ActorUsername,
		Action:        input.Action,
		ResourceType:  input.ResourceType,
		ResourceID:    input.ResourceID,
		Detail:        input.Detail,
		IP:            input.IP,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	return s.repo.CreateAuditEntry(entry)
}

func (s Service) List(limit int) ([]Entry, error) {
	if s.repo == nil {
		return nil, errors.New("audit repository is not configured")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListAuditEntries(limit)
}

// Prune deletes audit entries older than the configured retention window so the
// audit table does not grow without bound. The retention policy is the 90-day
// window advertised by Summary. A caller may invoke this once at startup and
// then on a periodic basis.
func (s Service) Prune() (int64, error) {
	if s.repo == nil {
		return 0, errors.New("audit repository is not configured")
	}
	cutoff := time.Now().UTC().Add(-90 * 24 * time.Hour)
	return s.repo.DeleteAuditEntriesBefore(cutoff)
}
