package backups

import (
	"errors"
	"strings"
)

type Summary struct {
	Targets      []string `json:"targets"`
	Capabilities []string `json:"capabilities"`
}

type Backup struct {
	ID         string `json:"id"`
	BackupKind string `json:"backupKind"`
	FilePath   string `json:"filePath"`
	Summary    string `json:"summary"`
	CreatedAt  string `json:"createdAt"`
}

type CreateInput struct {
	BackupKind string `json:"backupKind"`
	FilePath   string `json:"filePath"`
	Summary    string `json:"summary"`
}

type Repository interface {
	ListBackups() ([]Backup, error)
	CreateBackup(input CreateInput) (Backup, error)
	DeleteBackup(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		Targets:      []string{"database", "templates", "rule-sets", "settings", "xray-snapshots"},
		Capabilities: []string{"export", "restore", "retention", "audit-log"},
	}
}

func (s Service) List() ([]Backup, error) {
	if s.repo == nil {
		return nil, errors.New("backup repository is not configured")
	}
	return s.repo.ListBackups()
}

func (s Service) Create(input CreateInput) (Backup, error) {
	if s.repo == nil {
		return Backup{}, errors.New("backup repository is not configured")
	}
	if strings.TrimSpace(input.BackupKind) == "" {
		return Backup{}, errors.New("backup kind is required")
	}
	if strings.TrimSpace(input.FilePath) == "" {
		return Backup{}, errors.New("backup file path is required")
	}
	return s.repo.CreateBackup(input)
}

func (s Service) Delete(id string) error {
	if s.repo == nil {
		return errors.New("backup repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("backup id is required")
	}
	return s.repo.DeleteBackup(id)
}
