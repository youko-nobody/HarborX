package backups

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
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

type ExportInput struct {
	BackupKind string `json:"backupKind"`
	Summary    string `json:"summary"`
}

type Repository interface {
	ListBackups() ([]Backup, error)
	CreateBackup(input CreateInput) (Backup, error)
	DeleteBackup(id string) error
	ExportDatabaseBackup(filePath string) error
}

type Service struct {
	repo    Repository
	dataDir string
}

func NewService(repo Repository, dataDir string) Service {
	return Service{repo: repo, dataDir: dataDir}
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

func (s Service) ExportDatabase(input ExportInput) (Backup, error) {
	if s.repo == nil {
		return Backup{}, errors.New("backup repository is not configured")
	}
	if strings.TrimSpace(input.BackupKind) == "" {
		input.BackupKind = "database"
	}
	if input.BackupKind != "database" {
		return Backup{}, errors.New("only database export is implemented")
	}
	if strings.TrimSpace(input.Summary) == "" {
		input.Summary = "SQLite database export"
	}

	backupDir := filepath.Join(s.dataDir, "backups")
	fileName := fmt.Sprintf("harborx-db-%s.sqlite", time.Now().UTC().Format("20060102-150405"))
	filePath := filepath.Join(backupDir, fileName)
	if err := s.repo.ExportDatabaseBackup(filePath); err != nil {
		return Backup{}, err
	}
	return s.repo.CreateBackup(CreateInput{
		BackupKind: "database",
		FilePath:   filePath,
		Summary:    input.Summary,
	})
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
